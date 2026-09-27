package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPlayerLosesLifePublicDeclaration(t *testing.T) {
	player, amount := dsl.BindPlayer("loser"), dsl.BindNumber("lost")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerLosesLife(player).WithAmount(amount),
		dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
