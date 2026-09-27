package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPlayerDamageSourcePublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("damaged")
	source := dsl.BindDamageSource("source")
	card := dsl.NewCreature("Synthetic Damage Watcher", "", 1, 3, dsl.WithAbility(dsl.WhenEvent(
		dsl.DealsDamageToPlayer(dsl.This, player).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.FromPlayer(player), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
