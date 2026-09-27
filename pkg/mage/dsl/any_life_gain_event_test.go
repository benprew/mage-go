package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPlayerGainsLifePublicDeclaration(t *testing.T) {
	player, amount := dsl.BindPlayer("gainer"), dsl.BindNumber("gained")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerGainsLife(player).WithAmount(amount),
		dsl.LoseLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
