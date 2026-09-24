package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSourceCountersComposition(t *testing.T) {
	value := dsl.SourceCounters(core.Charge)
	n := dsl.BindNumber("amount")
	body := dsl.Sequence(dsl.SnapshotNumber(value, n), dsl.GainLife(dsl.You, dsl.FromNumber(n)))
	cards := []dsl.Card{
		dsl.NewEnchantment("Main phase", "", dsl.WithAbility(dsl.At(dsl.YourFirstMainPhase, dsl.AddManaAmount(core.Black, value)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Event", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
		dsl.NewCreature("Delayed", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.AtNext(dsl.EachEndStep, body)))),
		dsl.NewCreature("Modes", "", 1, 1, dsl.WithAbility(dsl.ActivateModes(dsl.Costs(), dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
