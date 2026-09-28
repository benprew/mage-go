package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCreatureEntryControllerPublicDeclaration(t *testing.T) {
	controller := dsl.BindPlayer("entrant controller")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.AnotherCreatureEnters().WithController(controller),
		dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
