package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextActivationLayerDeclaration(t *testing.T) {
	for _, kind := range []ActionKind{ActionSpell, ActionActivated} {
		card := NewArtifact("Declaration", "", WithAbility(NewAction(kind, WithLayeredTextRevisionActivationState())))
		err := ValidateCardDefinition(card.Name(), card)
		if (err == nil) != (kind == ActionActivated) {
			t.Fatalf("kind %v: %v", kind, err)
		}
	}
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedActivatedTextEffect(nil, uuid.New(), core.Indefinite, change); err == nil {
		t.Fatal("nil source accepted")
	}
}

func TestTextActivationLayerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "lost-marker", "missing-builder", "missing-group", "missing-history", "granted", "removed", "lost-limit", "changed-cost-shape", "changed-timing"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(layer, history, group, revised bool) CardFactory {
				return func() Card {
					options := []ActionOption{WithEffect(GainLifeFor(You, Fixed(1)))}
					if !revised || scenario != "lost-limit" {
						options = append(options, WithOncePerTurn())
					}
					if revised && scenario == "changed-cost-shape" {
						options = append(options, WithCost(GenericCost(1)))
					}
					if revised && scenario == "changed-timing" {
						options = append(options, WithSorcerySpeed())
					}
					if layer {
						options = append(options, WithLayeredTextRevisionActivationState())
					} else {
						options = append(options, WithTextRevisionActivationState())
					}
					if history {
						options = append(options, WithActivationHistory(BindActivationHistory("history")))
					}
					option := WithAbility(NewAction(ActionActivated, options...))
					if group {
						option = WithDefinitionAbilityGroup(1, option)
					}
					return NewArtifact("Declaration", "", option)
				}
			}
			plan, err := NewCardDefinitionPlan(build(scenario != "unmarked", true, scenario != "missing-group", false))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "missing-builder" {
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) {
					return build(scenario != "lost-marker", scenario != "missing-history", true, true), nil
				})(card.(*BaseCard))
			}
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(card, a.PlayerID())
			original := p.RuntimeAbilities[0]
			id := original.AbilityID()
			if scenario == "granted" {
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &grantedByEffect{Ability: original}
			}
			if scenario == "removed" {
				g.MutablePermanent(p.ID()).RuntimeAbilities = nil
			}
			before := g.FindPermanent(p.ID()).RuntimeAbilities
			count := len(g.effects.effects)
			change, _ := ReplaceColorWord(core.Red, core.Green)
			if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, change); err == nil {
				t.Fatal("incompatible declaration accepted")
			}
			after := g.FindPermanent(p.ID())
			if len(g.effects.effects) != count || len(after.layeredActivatedOrigins) != 0 || len(after.RuntimeAbilities) != len(before) {
				t.Fatal("rejection changed state")
			}
			for i := range before {
				if before[i] != after.RuntimeAbilities[i] {
					t.Fatal("rejection changed action")
				}
			}
		})
	}
}
