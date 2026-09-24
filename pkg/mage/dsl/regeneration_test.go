package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedRegenerationComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.AnyPermanent)
	saved := dsl.BindPermanent("saved")
	member := dsl.BindPermanent("member")
	for _, card := range []mage.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.Regenerate(target), dsl.WithTargets(target)))),
		dsl.NewCreature("Activation", "", 2, 2, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This)))),
		dsl.NewCreature("Trigger", "", 2, 2, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.Regenerate(dsl.From(saved)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), member, dsl.Regenerate(dsl.From(member)))))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Shield", dsl.Regenerate(target), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
