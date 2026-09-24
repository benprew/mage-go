package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCounterResetComposition(t *testing.T) {
	saved := dsl.BindPermanent("subject")
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.RemoveAllCounters(dsl.From(saved), dsl.Tide))
	for _, card := range []mage.Card{
		dsl.NewCreature("State", "", 1, 1, dsl.WithAbility(dsl.WhenState(dsl.SourceHasCounters(dsl.Tide, 4), body))),
		dsl.NewCreature("Activate", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body))),
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.RemoveAllCounters(target, dsl.Tide), dsl.WithTargets(target)))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.RemoveAllCounters(dsl.From(saved), dsl.Tide))))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Remove", dsl.RemoveAllCounters(target, dsl.Tide), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
