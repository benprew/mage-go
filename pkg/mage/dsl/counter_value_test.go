package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCounterValueComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	value := dsl.CountersOn(dsl.This, core.Charge)
	body := dsl.GainLife(dsl.You, value)
	cards := []dsl.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.GainLife(dsl.You, dsl.CountersOn(target, core.Charge)), dsl.WithTargets(target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Upkeep", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body))),
		dsl.NewCreature("Event", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewCreature("Sequence", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.GainLife(dsl.You, dsl.CountersOn(dsl.From(saved), core.Charge)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.GainLife(dsl.You, dsl.CountersOn(dsl.From(saved), core.Charge)))))),
		dsl.NewCreature("Modes", "", 1, 1, dsl.WithAbility(dsl.AtModes(dsl.YourUpkeep, dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Static", "", 1, 1, dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature), dsl.ModifyPT(dsl.This, value, dsl.Fixed(0)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
