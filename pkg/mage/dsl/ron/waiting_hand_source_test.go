package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingPrecastHandProtection(t *testing.T) {
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
		for _, state := range []string{"live", "departed", "returned", "copy", "clone"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
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
				paid := mage.BindCard("drawn hand source")
				keeper := mage.NewArtifact("Draw watcher", "", mage.WithAbility(mage.WhenEvent(mage.ControllerDrawsCardInto(mage.BindPlayer("drawer"), paid), mage.GainLifeFor(mage.You, mage.Fixed(1)))))
				g.PutOnBattlefield(keeper, a.PlayerID())
				card := mage.NewArtifact("Hand subject", map[core.Color]string{core.White: "{W}", core.Blue: "{U}", core.Black: "{B}", core.Red: "{R}", core.Green: "{G}"}[tc.color])
				a.SetLibrary([]mage.Card{card})
				g.PlayerDrawCard(chooser)
				g.PutTriggersOnStack()
				saved := g.StackHandSources()[0]
				seed := mage.BindDamageSource("waiting seed")
				if err := mage.ApplyEffect(g, mage.Sequence(mage.ChooseDamageSourceFromStackHandReferences(mage.You, mage.DamageSourceFilter{}, seed), mage.PreventNextDamageFromSelectedSource(mage.FromDamageSource(seed), mage.You, mage.DamageSourceFilter{})), uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.GetStack().Pop()
				if len(g.StackHandSources()) != 0 {
					t.Fatal("stack reference remained")
				}

				binding := mage.BindDamageSource("chosen hand source")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
				body := mage.Sequence(plan.waitingPrecastHandChoice(bindings), shield(bindings))
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
					a.RemoveFromHand(card.ID())
					if state == "returned" {
						a.AddToHand(card)
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
				if chooser.calls != 1 || !chooser.source.SameObject(saved) || chooser.source.HandVisit() == uuid.Nil {
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
				if err := mage.ApplyEffect(g, mage.Sequence(body, mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if a.Life() != want-1 {
					t.Fatal("empty category retained a stale choice")
				}
				if _, err := Compile(record); err != nil {
					t.Fatal("complete source-choice record rejected", err)
				}
			})
		}
	}
}

func TestRONWaitingPrecastHandInvalid(t *testing.T) {
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
	for _, effect := range []mage.ResolvingEffect{plan.waitingPrecastHandChoice(instanceBindings{}), mage.Sequence(mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)), plan.waitingPrecastHandChoice(bindings))} {
		if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), effect)))) == nil {
			t.Fatal("invalid binding accepted")
		}
	}
}
