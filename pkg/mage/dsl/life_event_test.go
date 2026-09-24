package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedLifeEventComposition(t *testing.T) {
	player, amount := dsl.BindPlayer("loser"), dsl.BindNumber("lost")
	pattern := dsl.OpponentLosesLife(player).WithAmount(amount)
	body := dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount))
	for _, card := range []mage.Card{
		dsl.NewEnchantment("Life", "", dsl.WithAbility(dsl.WhenEvent(pattern, body))),
		dsl.NewEnchantment("Modes", "", dsl.WithAbility(dsl.WhenEventModes(pattern,
			dsl.ModeOf("Gain", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := mage.InspectDefinition(body)
	if info.Role == "" {
		t.Fatal("missing definition metadata")
	}
}
