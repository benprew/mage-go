package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSkipNextUntapComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.SkipNextUntap(dsl.From(saved)))
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(dsl.SkipNextUntap(target), dsl.WithTargets(target)))),
		dsl.NewArtifact("Barl's Cage", "{4}", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{3}")), dsl.SkipNextUntap(target), dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Skip", dsl.SkipNextUntap(target), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.SkipNextUntap(dsl.From(saved)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
func TestCapturedUntapComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.SkipNextUntapOf(dsl.From(saved), dsl.You))
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(dsl.SkipNextUntapOf(target, dsl.You), dsl.WithTargets(target)))),
		dsl.NewArtifact("Barl's Cage", "{4}", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{3}")), dsl.SkipNextUntapOf(target, dsl.You), dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Skip", dsl.SkipNextUntapOf(target, dsl.You), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.SkipNextUntapOf(dsl.From(saved), dsl.You))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHomaridShroudActivationComposition(t *testing.T) {
	card := dsl.NewCreature("Homarid Warrior", "{4}{U}", 3, 3,
		dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{U}")),
			dsl.Sequence(
				dsl.Until(core.EndOfTurn, dsl.GrantKeyword(dsl.This, core.Shroud)),
				dsl.SkipNextUntapOf(dsl.This, dsl.You),
				dsl.Tap(dsl.This),
			),
		)),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
