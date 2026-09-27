package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDamageToControllerPublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	amount := dsl.BindNumber("damage dealt")
	card := dsl.NewAura("Damage Watcher", "{W}", dsl.WithAbility(
		dsl.WhenEvent(dsl.DealsDamageToPlayer(dsl.AttachedPermanent, player).ToController().WithAmount(amount),
			dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
