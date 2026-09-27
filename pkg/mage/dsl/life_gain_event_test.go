package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControllerLifeGainEventComposition(t *testing.T) {
	player := dsl.BindPlayer("gainer")
	var pattern = dsl.ControllerGainsLife(player)
	body := dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1))
	for _, card := range []mage.Card{
		dsl.NewEnchantment("Life", "", dsl.WithAbility(dsl.WhenEvent(pattern, body))),
		dsl.NewEnchantment("Modes", "", dsl.WithAbility(dsl.WhenEventModes(pattern,
			dsl.ModeOf("Gain", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	if info := mage.InspectDefinition(body); info.Properties.LifeGain != 1 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestControllerLifeGainAmountComposition(t *testing.T) {
	player := dsl.BindPlayer("gainer")
	amount := dsl.BindNumber("gained")
	card := dsl.NewEnchantment("Life Amount", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ControllerGainsLife(player).WithAmount(amount),
		dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
