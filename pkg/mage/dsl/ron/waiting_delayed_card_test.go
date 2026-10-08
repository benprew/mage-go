package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCardProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black},
		{"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green},
		{"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White},
		{"ATQ", "Circle of Protection: Artifacts", core.White},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		for _, zone := range []string{"hand", "exile"} {
			for _, state := range []string{"live", "departed", "returned", "copy", "clone"} {
				t.Run(tc.name+"/"+zone+"/"+state, func(t *testing.T) {
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
					_, a, b := ronGame()
					chooser := &ronSourceChoicePlayer{BasePlayer: a}
					g := mage.NewGame(chooser, b)
					nodes, err := Parse(strings.NewReader("CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([GainLife(Integer(1))]))"), "delay.ron")
					if err != nil {
						t.Fatal(err)
					}
					delay, err := compileDelayedStep(nodes[0], scope{})
					if err != nil {
						t.Fatal(err)
					}
					paid := mage.BindCard("saved source")
					card := mage.NewArtifact("Subject", map[core.Color]string{core.White: "{W}", core.Blue: "{U}", core.Black: "{B}", core.Red: "{R}", core.Green: "{G}"}[tc.color])
					card.SetOwner(a.PlayerID())
					var saved mage.DamageSource
					if zone == "hand" {
						keeper := mage.NewArtifact("Draw watcher", "", mage.WithAbility(mage.WhenEvent(mage.ControllerDrawsCardInto(mage.BindPlayer("drawer"), paid), delay(instanceBindings{}))))
						g.PutOnBattlefield(keeper, a.PlayerID())
						a.SetLibrary([]mage.Card{card})
						g.PlayerDrawCard(chooser)
						g.PutTriggersOnStack()
						saved = g.StackHandSources()[0]
					} else {
						keeper := mage.NewArtifact("Payment watcher", "", mage.WithAbility(mage.Activate(mage.Costs(mage.ExileMatchingCardFromGraveyardCostWithCard(mage.CardFilter{}, "Exile", paid)), delay(instanceBindings{}))))
						g.PutOnBattlefield(keeper, a.PlayerID())
						if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err == nil || len(g.DelayedTriggers()) != 0 {
							t.Fatal("failed payment installed delay")
						}
						a.AddToGraveyard(card)
						if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
							t.Fatal(err)
						}
						saved = g.StackBoundExileSources()[0]
					}
					g.ResolveTopOfStack()
					if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
						t.Fatal("producer did not transfer its card into the waiting frame")
					}

					binding := mage.BindDamageSource("chosen hand source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					body := mage.Sequence(plan.waitingDelayedCardChoice(bindings), shield(bindings))
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := mage.NewEnchantment("RON consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), body)))
					if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
						t.Fatal(err)
					}
					g.PutOnBattlefield(consumer, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err == nil || chooser.calls != 0 {
						t.Fatal("unpaid activation chose a source")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if chooser.calls != 0 {
						t.Fatal("choice ran at announcement")
					}
					switch state {
					case "departed", "returned":
						if zone == "hand" {
							a.RemoveFromHand(card.ID())
							if state == "returned" {
								a.AddToHand(card)
							}
						} else {
							g.RemoveFromExile(card.ID())
							if state == "returned" {
								g.ExileCard(card, uuid.New())
							}
						}
					case "copy":
						if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
					case "clone":
						clone := g.Clone()
						clone.ResolveTopOfStack()
						clone.DealDamageToPlayerFromSource(clone.GetPlayer(a.PlayerID()), 2, saved)
						if clone.GetPlayer(a.PlayerID()).Life() != 20 || chooser.calls != 0 {
							t.Fatalf("clone life=%d, original choice calls=%d, clone sources=%v", clone.GetPlayer(a.PlayerID()).Life(), chooser.calls, clone.WaitingPreventionHandSourcesFor(a.PlayerID()))
						}
					}
					g.ResolveTopOfStack()
					if chooser.calls != 1 || !chooser.source.SameObject(saved) {
						t.Fatal("choice lost exact RON hand reference")
					}
					if state == "copy" {
						g.ResolveTopOfStack()
						if chooser.calls != 2 {
							t.Fatal("copy did not choose separately")
						}
					}
					want := 20
					g.DealDamageToPlayerFromSource(chooser, 2, saved)
					if a.Life() != want {
						t.Fatal("saved reference lost prevention after stack departure")
					}
					if state == "copy" {
						g.DealDamageToPlayerFromSource(chooser, 2, saved)
					}
					g.DealDamageToPlayerFromSource(chooser, 1, saved)
					if a.Life() != want-1 {
						t.Fatal("shield consumption was not local")
					}
					g.DealDamageToPlayerFromSource(b, 1, saved)
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
					g.ResolveStack()
					if err := mage.ApplyEffect(g, mage.Sequence(body, mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if a.Life() != want {
						t.Fatal("empty category retained a stale choice")
					}
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedCardInvalid(t *testing.T) {
	for _, text := range []string{"ChooseADamageSource()", "ChooseADamageSource(IsColor(Colorless))", "ChooseADamageSource(And([]))", "ChooseADamageSource(Ref_TargetPermanent)"} {
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	plan := &chosenDamageSourcePlan{}
	binding := mage.BindDamageSource("waiting hand")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	for _, effect := range []mage.ResolvingEffect{plan.waitingDelayedCardChoice(instanceBindings{}), mage.Sequence(mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)), plan.waitingDelayedCardChoice(bindings))} {
		if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), effect)))) == nil {
			t.Fatal("invalid binding accepted")
		}
	}
}
