package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOpponentCreatureDiesAuthorExample(t *testing.T) {
	former := dsl.BindPlayer("former controller")
	ability := dsl.WhenEvent(
		dsl.OpponentCreatureDies().WithController(former),
		dsl.GainLife(dsl.FromPlayer(former), dsl.Fixed(1)),
	)
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
