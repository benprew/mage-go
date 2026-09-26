package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCombatDamageEventAuthorSurface(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	amount := dsl.BindNumber("combat damage")
	var card dsl.Card = dsl.NewCreature("Combat Watcher", "{2}{G}", 2, 2,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.DealsCombatDamageToPlayer(dsl.This, player).WithAmount(amount),
			dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
