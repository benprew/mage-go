package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDamageSourceControllerPublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	controller := dsl.BindPlayer("source controller")
	amount := dsl.BindNumber("damage dealt")
	card := dsl.NewAura("Damage Watcher", "{U}", dsl.WithAbility(
		dsl.WhenEvent(
			dsl.DealsDamageToPlayer(dsl.AttachedPermanent, player).ToController().WithSourceController(controller).WithAmount(amount),
			dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.FromNumber(amount)),
		),
	))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
