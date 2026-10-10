package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextConditionalPTSpellDeclaration(t *testing.T) {
	for _, effect := range []ContinuousEffect{
		SourcePTWhileControllerControlsBasicLand("Forest", 1, 2),
		SourcePTWhileOpponentControlsBasicLand("Island", 0, 1),
	} {
		card := NewCreature("Conditional", "", 1, 1, WithGlobalStaticSpellTextContinuation(), WithAbility(StaticAbility(effect).WithLayeredPTTextRevision()))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, effect := range []ContinuousEffect{
		SourcePTWhileControllerControlsBasicLand("Desert", 1, 2),
		SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1),
		NamedCreaturesPTWhileOpponentControlsNontokenColor("Conditional", core.Red, 1, 1),
		CreaturesPTWhileAttachedToBasicLand("Forest", "Goblin", 1, 1),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithGlobalStaticSpellTextContinuation(), WithAbility(StaticAbility(effect).WithLayeredPTTextRevision()))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("unsupported context accepted")
		}
	}
}

func TestTextConditionalPTSpellRejectsContextAtomic(t *testing.T) {
	for _, kind := range []string{"opponent", "global", "color", "extra", "unmarked"} {
		t.Run(kind, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					effect := SourcePTWhileControllerControlsBasicLand("Forest", 1, 2)
					if changed {
						switch kind {
						case "opponent":
							effect = SourcePTWhileOpponentControlsBasicLand("Island", 1, 2)
						case "global":
							effect = BoostAllCreaturesIncludingSelf(1, 2, IsCreature)
						case "color":
							effect = SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 2)
						}
					}
					holder := StaticAbility(effect).WithLayeredPTTextRevision()
					if changed && kind == "unmarked" {
						holder = StaticAbility(effect)
					}
					opts := []CardOption{WithGlobalStaticSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithAbility(holder))}
					if changed && kind == "extra" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					return NewCreature("Conditional", "", 1, 1, opts...)
				}
			}
			plan, err := NewCardDefinitionPlan(build(false))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			g, a, _ := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			change, _ := ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
			if err := g.RevisePrintedSpellText(before.ID, change); err == nil {
				t.Fatal("changed context accepted")
			}
			if g.StackPeek() != before {
				t.Fatal("rejection changed stack")
			}
		})
	}
}
