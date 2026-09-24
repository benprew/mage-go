package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedCounterAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount")
	counters := dsl.AddCounters(dsl.From(saved), dsl.P1P1, dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), counters)
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.AddCounters(target, dsl.P1P1, dsl.ToughnessOf(target)), dsl.WithTargets(target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, counters)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Counters", dsl.AddCounters(target, dsl.P1P1, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
