package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControllerDrawEventComposition(t *testing.T) {
	player := dsl.BindPlayer("drawer")
	var pattern = dsl.ControllerDrawsCard(player)
	body := dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1))
	for _, card := range []mage.Card{
		dsl.NewEnchantment("Draw", "", dsl.WithAbility(dsl.WhenEvent(pattern, body))),
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
