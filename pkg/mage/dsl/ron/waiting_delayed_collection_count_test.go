package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCollectionCountProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black},
		{"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green},
		{"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White},
		{"ATQ", "Circle of Protection: Artifacts", core.Colorless},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		for _, shape := range []string{"singleton", "attachments", "blockers", "blocked", "history", "owner"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy", "clone"} {
				t.Run(tc.name+"/"+shape+"/"+state, func(t *testing.T) {
					record := staticBaseRecord(t, tc.set, tc.name)
					rule := record.Fields["Rules"].Args[0]
					actions := rule.Args[1].Args[0].Args
					plan, err := compileChosenDamageSourceChoice(actions[0])
					if err != nil {
						t.Fatal(err)
					}
					shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
					if err != nil {
						t.Fatal(err)
					}
					costs, _, err := compileCosts(rule.Args[0])
					if err != nil {
						t.Fatal(err)
					}
					a := &ronDelayedPaymentPlayer{ronSourceChoicePlayer: &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
					b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
					g := mage.NewGame(a, b)
					values := map[string]string{
						"singleton":   `TheNumberOfPermanentsOnTheBattlefield(SinglePermanent(ThisPermanent))`,
						"attachments": `TheNumberOfPermanentsOnTheBattlefield(AttachedToAPermanent(SinglePermanent(ThisPermanent)))`,
						"blockers":    `TheNumberOfPermanentsOnTheBattlefield(IsBlockingAttacker(ThisPermanent))`,
						"blocked":     `TheNumberOfPermanentsOnTheBattlefield(IsBlockedByDefender(ThisPermanent))`,
						"history":     `TheNumberOfPermanentsOnTheBattlefield(WasBlockedByDefenderThisTurn(ThisPermanent))`,
						"owner":       `TheNumberOfPermanentsOnTheBattlefield(And([IsCardtype(Creature),OwnedByAPlayer(SinglePlayer(ControllerOfPermanent(ThisPermanent)))]))`,
					}
					body := `PermanentDealsDamage(Ref_TargetPermanent,` + values[shape] + `,Player(You))`
					factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))`)
					for _, name := range []string{"B1", "B2"} {
						creature := mage.NewCreature(name, "", 1, 4)
						creature.SetOwner(b.PlayerID())
						g.PutOnBattlefield(creature, b.PlayerID())
					}
					donorCard := mage.NewCreature("Donor", "", 1, 3)
					donorCard.SetOwner(a.PlayerID())
					donor := g.PutOnBattlefield(donorCard, a.PlayerID())
					card := factory()
					card.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(card, a.PlayerID())
					g.ChangeObjectColor(source.ID(), tc.color)
					original := g.CaptureDamageSource(source.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
						t.Fatal(err)
					}
					if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("delay created at announcement")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(source)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					binding := mage.BindDamageSource("waiting player source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
					p := g.PutOnBattlefield(consumer, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 || a.calls != 0 {
						t.Fatal("failed payment changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if a.calls != 0 || a.offers != 0 {
						t.Fatal("choice ran at announcement")
					}
					a.ManaPool().Clear()

					clone := g.Clone()
					if state == "clone" {
						clone.ResolveStack()
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						clone.PutTriggersOnStack()
						clone.ResolveStack()
						if len(g.WaitingDelayedPermanentSources()) != 2 || a.calls != 0 || a.offers != 0 || a.Life() != 20 {
							t.Fatal("clone changed original")
						}
						wantClone := 20 - collectionCountExpected(shape, state)
						if clone.GetPlayer(a.PlayerID()).Life() != wantClone || len(clone.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("clone lost controller read or waiting lifetime")
						}
					}
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						found := false
						for _, candidate := range got {
							if candidate.SameObject(original) {
								found = true
							}
						}
						if len(got) != 2 || !found || a.offers != 0 || b.offers != 0 {
							t.Fatal("query lost source or offered payment")
						}
					}
					g.ResolveStack()
					if a.calls != 1 || !a.source.SameObject(original) || a.offers != 0 || b.offers != 0 {
						t.Fatal("typed player read did not supply source choice")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("detected delay still waits")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					damage := collectionCountExpected(shape, state)
					if state == "copy" {
						damage *= 2
					}
					wantA, wantB := 20-damage, 20
					if a.Life() != wantA || b.Life() != wantB || a.offers != 0 || b.offers != 0 {
						t.Fatal("wrong current or departure damage operand", a.Life(), b.Life())
					}
					g.DealDamageToPlayerFromSource(a, 2, original)
					if a.Life() != wantA {
						t.Fatal("selected source damage not prevented")
					}
					g.DealDamageToPlayerFromSource(a, 1, original)
					if a.Life() != wantA-1 {
						t.Fatal("shield not consumed")
					}
					if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
						t.Fatal("incomplete aggregation accepted", err)
					}
				})
			}
		}
	}
}

func collectionCountExpected(shape, state string) int {
	if shape == "owner" {
		if state == "control" {
			return 2
		}
		return 1
	}
	if shape == "singleton" && state != "departed" && state != "returned" && state != "phased" {
		return 1
	}
	return 0
}

func TestRONWaitingDelayedCollectionCountComposition(t *testing.T) {
	for _, tc := range []struct {
		set  string
		want int
	}{
		{`And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(You))])`, 1},
		{`Or([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))])`, 2},
		{`Or([SinglePermanent(ThisPermanent),SinglePermanent(ThisPermanent)])`, 1},
		{`And([SinglePermanent(ThisPermanent),IsCardtype(Creature)])`, 0},
		{`AttachedToAPermanent(SinglePermanent(ThisPermanent))`, 1},
	} {
		t.Run(tc.set, func(t *testing.T) {
			g, a, b := ronGame()
			f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(TheNumberOfPermanentsOnTheBattlefield(`+tc.set+`))]))]))`)
			p := g.PutOnBattlefield(f(), a.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 4), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			attachment := g.PutOnBattlefield(mage.NewEnchantment("Late attachment", "", mage.WithSubTypes("Aura")), b.PlayerID())
			g.Attach(attachment.ID(), p.ID())
			for range 3 {
				if a.Life() != 20 {
					t.Fatal("query executed body")
				}
				g.WaitingDelayedPermanentSources()
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 20+tc.want {
					t.Fatal("wrong composed membership", game.GetPlayer(a.PlayerID()).Life(), tc.want)
				}
			}
		})
	}
}

func TestRONWaitingDelayedCollectionCountInvalid(t *testing.T) {
	for _, set := range []string{
		`SinglePermanent()`, `SinglePermanent(Trigger_ThatPermanent)`, `SinglePermanent(Ref_TargetPermanent)`,
		`SinglePermanent(You)`, `AttachedToAPermanent()`, `AttachedToAPermanent(SinglePermanent(Ref_TargetPlayer))`,
		`IsBlockingAttacker()`, `IsBlockedByDefender(Trigger_ThatCreature)`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent)`,
		`OwnedByAPlayer(SinglePlayer(Trigger_ThatPlayer))`, `And([])`, `Or([])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(TheNumberOfPermanentsOnTheBattlefield(`+set+`))]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid collection accepted", set)
		}
	}
}

func TestRONWaitingDelayedCollectionCountInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(TheNumberOfPermanentsOnTheBattlefield(Or([SinglePermanent(ThisPermanent),SinglePermanent(Ref_TargetPermanent)])))]))])))`)
	for _, player := range []*mage.BasePlayer{a, b} {
		target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 4), player.PlayerID())
		source := g.PutOnBattlefield(factory(), player.PlayerID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if player == a {
			g.BouncePermanentToHand(source)
			a.RemoveFromHand(source.ID())
			g.PutOnBattlefield(source.Card, b.PlayerID())
			g.BouncePermanentToHand(target)
			a.RemoveFromHand(target.ID())
			g.PutOnBattlefield(target.Card, b.PlayerID())
		} else {
			g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
		}
	}
	if got := g.WaitingDelayedPermanentSources(); len(got) != 4 {
		t.Fatal("instances lost source or target identity", len(got))
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatal("returned objects or later shroud changed saved collection", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}
