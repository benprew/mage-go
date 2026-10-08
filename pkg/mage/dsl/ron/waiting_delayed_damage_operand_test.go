package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedDamageOperandProtection(t *testing.T) {
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
		for _, shape := range []string{"recipient", "value"} {
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
					body := `PermanentDealsDamage(Ref_TargetPermanent,Integer(1),Player(ControllerOfPermanent(ThisPermanent)))`
					if shape == "value" {
						body = `PermanentDealsDamage(Ref_TargetPermanent,TheNumberOfCardsInPlayersHand(ControllerOfPermanent(ThisPermanent)),Player(You))`
					}
					factory := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))`)
					a.AddToHand(mage.NewArtifact("Hand A", ""))
					b.AddToHand(mage.NewArtifact("Hand B", ""))
					b.AddToHand(mage.NewArtifact("Hand B2", ""))
					donor := g.PutOnBattlefield(mage.NewCreature("Donor", "", 1, 3), a.PlayerID())
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
						wantClone := 19
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
					damage := 1
					if state == "copy" {
						damage = 2
					}
					wantA, wantB := 20-damage, 20
					if state == "control" {
						if shape == "recipient" {
							wantA, wantB = 20, 19
						} else {
							wantA = 18
						}
					}
					if state == "departed" && shape == "value" {
						wantA = 18
					}
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
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedDamageOperandInvalid(t *testing.T) {
	for _, body := range []string{
		`PermanentDealsDamage(ThisPermanent,Integer(1),Player(ControllerOfPermanent()))`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Player(ControllerOfPermanent(Ref_TargetPermanent)))`,
		`PermanentDealsDamage(ThisPermanent,TheNumberOfCardsInPlayersHand(Trigger_ThatPlayer),Player(You))`,
		`PermanentDealsDamage(ThisPermanent,TheNumberOfCardsInPlayersHand(ControllerOfPermanent(ThisPermanent,Integer(1))),Player(You))`,
		`PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Player(ControllerOfPermanent(Trigger_ThatPermanent)))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid damage operand accepted", body)
		}
	}
}

func TestRONWaitingDelayedDamageOperandInstances(t *testing.T) {
	for _, amount := range []string{
		`TheNumberOfCardsInPlayersHand(ControllerOfPermanent(ThisPermanent))`,
		`DamageDealtToPlayerThisTurn(ControllerOfPermanent(ThisPermanent))`,
		`DamageDealtToPlayerThisTurnByPermanents(ControllerOfPermanent(ThisPermanent),IsCardtype(Artifact))`,
	} {
		t.Run(amount, func(t *testing.T) {
			g, a, b := ronGame()
			for range 2 {
				a.AddToHand(mage.NewArtifact("Hand A", ""))
			}
			for range 3 {
				b.AddToHand(mage.NewArtifact("Hand B", ""))
			}
			donorCard := mage.NewCreature("Damage", "", 1, 4)
			donorCard.AddType(core.TypeArtifact)
			donor := g.PutOnBattlefield(donorCard, a.PlayerID())
			g.DealDamageToPlayerFromSource(a, 2, g.CaptureDamageSource(donor.ID()))
			g.DealDamageToPlayerFromSource(b, 3, g.CaptureDamageSource(donor.ID()))
			factory := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PermanentDealsDamage(Ref_TargetPermanent,`+amount+`,Player(You))]))])))`)
			saved := make([]mage.DamageSource, 0, 2)
			for _, player := range []*mage.BasePlayer{a, b} {
				card := factory()
				card.SetOwner(player.PlayerID())
				p := g.PutOnBattlefield(card, player.PlayerID())
				saved = append(saved, g.CaptureDamageSource(p.ID()))
				if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.DestroyPermanent(p)
			}
			got := g.WaitingDelayedPermanentSources()
			found := 0
			for _, source := range got {
				for _, original := range saved {
					if source.SameObject(original) {
						found++
					}
				}
			}
			if len(got) != 3 || found != 2 {
				t.Fatal("instances shared source identity")
			}
			if a.Life() != 18 || b.Life() != 17 {
				t.Fatal("query read and executed value")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 16 || game.GetPlayer(b.PlayerID()).Life() != 14 || len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("per-instance value or controller lost")
				}
			}
		})
	}
}
