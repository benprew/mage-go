package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedUntapProtection(t *testing.T) {
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
		for _, shape := range []string{"single", "counted", "zero", "singleton", "attachments", "empty", "owner"} {
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
					bodies := map[string]string{
						"single":      `PermanentDoesntUntapDuringControllersNextUntap(ThisPermanent)`,
						"counted":     `PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent,Integer(2))`,
						"zero":        `PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent,Integer(0))`,
						"singleton":   `EachPermanentDoesntUntapDuringControllersNextUntap(Or([SinglePermanent(ThisPermanent),SinglePermanent(ThisPermanent)]))`,
						"attachments": `EachPermanentDoesntUntapDuringControllersNextUntap(AttachedToAPermanent(SinglePermanent(ThisPermanent)))`,
						"empty":       `EachPermanentDoesntUntapDuringControllersNextUntap(And([SinglePermanent(ThisPermanent),IsCardtype(Land)]))`,
						"owner":       `EachPermanentDoesntUntapDuringControllersNextUntap(And([IsCardtype(Creature),OwnedByAPlayer(SinglePlayer(ControllerOfPermanent(ThisPermanent)))]))`,
					}
					body := bodies[shape]
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
					binding := mage.BindDamageSource("waiting untap source")
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
						wantClone := 20
						if clone.GetPlayer(a.PlayerID()).Life() != wantClone || len(clone.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("clone lost waiting lifetime")
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
						t.Fatal("untap source read did not supply source choice")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("detected delay still waits")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					if a.Life() != 20 || b.Life() != 20 || a.offers != 0 || b.offers != 0 {
						t.Fatal("untap query changed life or offered payment")
					}
					for step := range 3 {
						for _, player := range []*mage.BasePlayer{a.BasePlayer, b.BasePlayer} {
							for _, p := range g.FilterBattlefield(mage.AnyPermanent) {
								g.TapPermanent(p)
							}
							if player == a.BasePlayer {
								g.SetActivePlayerIndex(0)
							} else {
								g.SetActivePlayerIndex(1)
							}
							g.RunStepWithPriority(core.Untap)
							for _, p := range g.FilterBattlefield(mage.AnyPermanent) {
								if p.ControllerID() != player.PlayerID() {
									continue
								}
								restricted := false
								switch shape {
								case "owner":
									restricted = p.Name() == "Donor"
									if state == "control" {
										restricted = p.Name() == "B1" || p.Name() == "B2"
									}
								case "single", "singleton", "counted":
									restricted = p.ID() == source.ID() && (state == "live" || state == "control" || state == "copy" || state == "clone")
								}
								count := 1
								if shape == "counted" {
									count = 2
								}
								if p.Tapped != (restricted && step < count) {
									t.Fatalf("wrong untap restriction: %s step %d", p.Name(), step)
								}
							}
						}
					}
					wantA := 20
					g.DealDamageToPlayerFromSource(a, 2, original)
					if a.Life() != wantA {
						t.Fatal("selected source damage not prevented")
					}
					g.DealDamageToPlayerFromSource(a, 1, original)
					if a.Life() != wantA-1 {
						t.Fatal("shield not consumed")
					}
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedUntapInvalid(t *testing.T) {
	for _, action := range []string{
		`PermanentDoesntUntapDuringControllersNextUntap()`,
		`PermanentDoesntUntapDuringControllersNextUntap(You)`,
		`PermanentDoesntUntapDuringControllersNextUntap(Trigger_ThatPermanent)`,
		`PermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPermanent)`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent,Integer(-1))`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent,ValueX)`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(And([]))`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(Or([]))`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPermanents)`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(SinglePermanent(Ref_TargetPlayer))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed untap accepted", action)
		}
	}
}

func TestRONWaitingDelayedUntapInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPermanentDoesntUntapDuringControllersNextUntap(Or([SinglePermanent(ThisPermanent),SinglePermanent(Ref_TargetPermanent)]))]))])))`)
	for _, player := range []*mage.BasePlayer{a, b} {
		card := mage.NewCreature("Target", "", 1, 4)
		card.SetOwner(player.PlayerID())
		target := g.PutOnBattlefield(card, player.PlayerID())
		sourceCard := factory()
		sourceCard.SetOwner(player.PlayerID())
		source := g.PutOnBattlefield(sourceCard, player.PlayerID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if player == a {
			for _, p := range []*mage.Permanent{source, target} {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(p.ID())
				g.PutOnBattlefield(p.Card, b.PlayerID())
			}
		} else {
			g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 4 {
		t.Fatal("instances lost exact source or saved target")
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		if len(game.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("detected delay still waits")
		}
		game.PutTriggersOnStack()
		game.ResolveStack()
		for _, p := range game.FilterBattlefield(mage.AnyPermanent) {
			game.TapPermanent(p)
		}
		game.SetActivePlayerIndex(1)
		game.RunStepWithPriority(core.Untap)
		for _, p := range game.FilterBattlefield(mage.AnyPermanent) {
			if p.Tapped != (p.Card.Owner() == b.PlayerID()) {
				t.Fatal("saved subject followed return or rechecked later shroud")
			}
		}
		game.RunStepWithPriority(core.Untap)
		for _, p := range game.FilterBattlefield(mage.AnyPermanent) {
			if p.Tapped {
				t.Fatal("restriction did not expire")
			}
		}
	}
}

func TestRONWaitingDelayedUntapLateAttachment(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPermanentDoesntUntapDuringControllersNextUntap(AttachedToAPermanent(SinglePermanent(ThisPermanent)))]))]))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("empty collection lost host declaration")
	}
	attachment := g.PutOnBattlefield(mage.NewEnchantment("Late Aura", "", mage.WithSubTypes("Aura")), b.PlayerID())
	g.Attach(attachment.ID(), source.ID())
	clone := g.Clone()
	clone.PhaseOut(attachment.ID())
	for _, game := range []*mage.Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		game.SetActivePlayerIndex(1)
		game.TapPermanent(game.FindPermanent(attachment.ID()))
		game.RunStepWithPriority(core.Untap)
	}
	if !g.FindPermanent(attachment.ID()).Tapped || clone.FindPermanent(attachment.ID()).Tapped {
		t.Fatal("wrong resolution membership or phase exclusion")
	}
}
