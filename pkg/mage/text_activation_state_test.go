package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextActivationStateDeclaration(t *testing.T) {
	for _, kind := range []ActionKind{ActionSpell, ActionActivated} {
		card := NewArtifact("Declaration", "", WithAbility(NewAction(kind, WithTextRevisionActivationState())))
		err := ValidateCardDefinition(card.Name(), card)
		if (err == nil) != (kind == ActionActivated) {
			t.Fatalf("kind %v: %v", kind, err)
		}
	}
}

func TestTextActivationStateMissingSource(t *testing.T) {
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedActivatedAbility(nil, uuid.New(), change); err == nil {
		t.Fatal("nil source accepted")
	}
	source := NewPermanent(NewArtifact("Absent", ""), g.players[0].PlayerID())
	if err := g.RevisePrintedActivatedAbility(source, uuid.New(), change); err == nil {
		t.Fatal("absent source accepted")
	}
}

func TestTextActivationStateContractMismatch(t *testing.T) {
	for _, scenario := range []string{"missing-history", "unmarked", "grant", "face-down"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(history, marked bool) CardFactory {
				return func() Card {
					options := []ActionOption{WithEffect(GainLifeFor(You, Fixed(1)))}
					if marked {
						options = append(options, WithTextRevisionActivationState())
					}
					if history {
						options = append(options, WithActivationHistory(BindActivationHistory("history")))
					}
					return NewArtifact("State", "", WithDefinitionAbilityGroup(1, WithAbility(NewAction(ActionActivated, options...))))
				}
			}
			plan, err := NewCardDefinitionPlan(build(true, scenario != "unmarked"))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(scenario != "missing-history", true), nil })(card.(*BaseCard))
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(card, a.PlayerID())
			original := p.RuntimeAbilities[0].(*ActionDefinition)
			if scenario == "grant" {
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &grantedByEffect{Ability: original}
			}
			if scenario == "face-down" {
				g.MutablePermanent(p.ID()).FaceDown = true
			}
			before := g.FindPermanentIncludingPhased(p.ID()).RuntimeAbilities[0]
			change, _ := ReplaceColorWord(core.Red, core.Green)
			if err := g.RevisePrintedActivatedAbility(p, original.AbilityID(), change); err == nil {
				t.Fatal("invalid state contract accepted")
			}
			if g.FindPermanentIncludingPhased(p.ID()).RuntimeAbilities[0] != before {
				t.Fatal("rejected revision mutated state")
			}
		})
	}
}
