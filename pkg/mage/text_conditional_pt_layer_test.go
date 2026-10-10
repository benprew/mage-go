package mage

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextConditionalPTLayerDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		effects []ContinuousEffect
		valid   bool
	}{
		{"controller", []ContinuousEffect{SourcePTWhileControllerControlsBasicLand("Forest", 1, 2)}, true},
		{"opponent-land", []ContinuousEffect{SourcePTWhileOpponentControlsBasicLand("Island", 0, 1)}, true},
		{"opponent-color", []ContinuousEffect{SourcePTWhileOpponentControlsNontokenColor(core.White, 1, 1)}, true},
		{"opponent-named", []ContinuousEffect{NamedCreaturesPTWhileOpponentControlsNontokenColor("Member", core.Red, 1, 1)}, true},
		{"host", []ContinuousEffect{CreaturesPTWhileAttachedToBasicLand("Mountain", "Goblin", 0, 2)}, true},
		{"mixed", []ContinuousEffect{SourcePTWhileControllerControlsBasicLand("Forest", 1, 2), BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)}, false},
		{"invalid-land", []ContinuousEffect{SourcePTWhileControllerControlsBasicLand("Desert", 1, 2)}, false},
		{"invalid-color", []ContinuousEffect{SourcePTWhileOpponentControlsNontokenColor(core.Colorless, 1, 1)}, false},
		{"empty-name", []ContinuousEffect{NamedCreaturesPTWhileOpponentControlsNontokenColor("", core.Red, 1, 1)}, false},
		{"invalid-host", []ContinuousEffect{CreaturesPTWhileAttachedToBasicLand("Mountain", "Aura", 0, 2)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewEnchantment("Rule", "", WithAbility(StaticAbility(tc.effects...).WithLayeredPTTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != tc.valid {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}

func TestTextConditionalPTLayerContextRejects(t *testing.T) {
	original := func() ContinuousEffect {
		return NamedCreaturesPTWhileOpponentControlsNontokenColor("Member", core.Red, 1, 1)
	}
	for _, next := range []func() ContinuousEffect{
		func() ContinuousEffect {
			return NamedCreaturesPTWhileOpponentControlsNontokenColor("Other", core.Blue, 1, 1)
		},
		func() ContinuousEffect { return SourcePTWhileOpponentControlsNontokenColor(core.Blue, 1, 1) },
		func() ContinuousEffect { return SourcePTWhileOpponentControlsBasicLand("Forest", 1, 1) },
		func() ContinuousEffect { return SourcePTWhileControllerControlsBasicLand("Forest", 1, 1) },
		func() ContinuousEffect { return CreaturesPTWhileAttachedToBasicLand("Mountain", "Goblin", 1, 1) },
		func() ContinuousEffect { return BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent) },
	} {
		build := func(effect func() ContinuousEffect) CardFactory {
			return func() Card {
				return NewEnchantment("Rule", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(effect()).WithLayeredPTTextRevision())))
			}
		}
		plan, err := NewCardDefinitionPlan(build(original))
		if err != nil {
			t.Fatal(err)
		}
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(next), nil })(card.(*BaseCard))
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		source := g.PutOnBattlefield(card, a.PlayerID())
		before := slices.Clone(g.effects.effects)
		abilities := slices.Clone(source.RuntimeAbilities)
		change, _ := ReplaceColorWord(core.Red, core.Blue)
		if err := g.AddPrintedStaticPTTextEffect(source, source.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err == nil {
			t.Fatal("changed condition or recipients accepted")
		}
		if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, g.FindPermanent(source.ID()).RuntimeAbilities) || len(source.layeredStaticAbilityOrigins) != 0 {
			t.Fatal("rejection changed live state")
		}
	}
}

func TestTextConditionalPTLayerHostContextRejects(t *testing.T) {
	build := func(creatureType string) CardFactory {
		return func() Card {
			return NewAura("Rule", "", WithCastTarget(TargetPermanent(IsLand)), WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(CreaturesPTWhileAttachedToBasicLand("Mountain", creatureType, 1, 2)).WithLayeredPTTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build("Goblin"))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build("Elf"), nil })(card.(*BaseCard))
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(card, a.PlayerID())
	before := slices.Clone(g.effects.effects)
	abilities := slices.Clone(source.RuntimeAbilities)
	change, _ := ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	if err := g.AddPrintedStaticPTTextEffect(source, source.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err == nil {
		t.Fatal("changed host recipient subtype accepted")
	}
	if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, g.FindPermanent(source.ID()).RuntimeAbilities) || len(source.layeredStaticAbilityOrigins) != 0 {
		t.Fatal("rejection changed live state")
	}
}
