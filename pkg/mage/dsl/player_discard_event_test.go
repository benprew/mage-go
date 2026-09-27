package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPlayerDiscardsCardAuthorExample(t *testing.T) {
	discarder := dsl.BindPlayer("discarder")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerDiscardsCard(discarder),
		dsl.GainLife(dsl.FromPlayer(discarder), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
