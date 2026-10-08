package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHandTargetSourcesProtection(t *testing.T) {
	for _, tc := range []struct{ set, name, mana string }{
		{"LEB", "Circle of Protection: Black", "{B}"},
		{"LEA", "Circle of Protection: Blue", "{U}"},
		{"LEA", "Circle of Protection: Green", "{G}"},
		{"LEA", "Circle of Protection: Red", "{R}"},
		{"LEA", "Circle of Protection: White", "{W}"},
		{"ATQ", "Circle of Protection: Artifacts", "{W}"},
		{"LEG", "Greater Realm of Preservation", "{B}"},
	} {
		for _, state := range []string{"live", "departed", "returned", "copy", "clone", "keeper phased", "keeper departed"} {
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
				card := mage.NewArtifact("Private target source", tc.mana)
				a.AddToHand(card)
				keeper := mage.NewArtifact("Hand target keeper", "", mage.WithAbility(mage.NewActivated(mage.GenericCost(0), mage.GainLifeFor(mage.You, mage.Fixed(1)), mage.WithTarget(mage.TargetCardInHand()))))
				g.PutOnBattlefield(keeper, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, []uuid.UUID{card.ID()}); err != nil {
					t.Fatal(err)
				}
				if len(g.StackHandSources()) != 1 || g.StackHandSources()[0].Name() != "" {
					t.Fatal("private hand target was public or absent")
				}
				saved := g.StackHandSources()[0]
				binding := mage.BindDamageSource("private source choice")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
				body := mage.Sequence(plan.inspectableStackHandChoice(bindings), shield(bindings))
				payment := make([]mage.Cost, len(costs))
				for i, cost := range costs {
					payment[i] = cost()
				}
				consumer := mage.NewEnchantment("RON inspection consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), body)))
				if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
					t.Fatal(err)
				}
				g.PutOnBattlefield(consumer, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err == nil || chooser.calls != 0 {
					t.Fatal("unpaid activation inspected a source")
				}
				a.ManaPool().Add(core.White, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if chooser.calls != 0 {
					t.Fatal("inspection ran at announcement")
				}
				switch state {
				case "keeper phased":
					g.PhaseOut(keeper.ID())
				case "keeper departed":
					g.DoSacrifice(g.FindPermanent(keeper.ID()))
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
						t.Fatal("clone private inspection failed")
					}
				}
				g.ResolveTopOfStack()
				if chooser.calls != 1 || !chooser.source.SameObject(saved) || chooser.source.Name() != card.Name() {
					t.Fatal("RON choice lost private information")
				}
				if state == "copy" {
					g.ResolveTopOfStack()
					if chooser.calls != 2 {
						t.Fatal("copy reused original choice")
					}
				}
				if g.StackHandSources()[0].Name() != "" {
					t.Fatal("private choice revealed hand")
				}
				g.GetStack().Pop()
				g.DealDamageToPlayerFromSource(chooser, 2, saved)
				if a.Life() != 20 {
					t.Fatal("private source lost shield after stack departure")
				}
				if state == "copy" {
					g.DealDamageToPlayerFromSource(chooser, 2, saved)
				}
				g.DealDamageToPlayerFromSource(chooser, 1, saved)
				if a.Life() != 19 {
					t.Fatal("shield did not consume")
				}
				if _, err := Compile(record); err != nil {
					t.Fatal("complete source-choice record rejected", err)
				}
			})
		}
	}
}

func TestRONHandTargetSourcesInvalid(t *testing.T) {
	for _, text := range []string{"ChooseADamageSource()", "ChooseADamageSource(IsColor(Colorless))", "ChooseADamageSource(And([]))", "ChooseADamageSource(Ref_TargetPermanent)"} {
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid private choice accepted")
		}
	}
	plan := &chosenDamageSourcePlan{}
	binding := mage.BindDamageSource("hand target source")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	bad := mage.Sequence(mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)), plan.inspectableStackHandChoice(bindings))
	if mage.ValidateCardDefinition("Invalid order", mage.NewArtifact("Invalid order", "", mage.WithAbility(mage.Activate(mage.Costs(), bad)))) == nil {
		t.Fatal("premature choice read accepted")
	}
	if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), plan.inspectableStackHandChoice(instanceBindings{}))))) == nil {
		t.Fatal("missing instance binding accepted")
	}
}
