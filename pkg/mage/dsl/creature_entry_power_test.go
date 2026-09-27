package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCreatureEntryPowerPublicDeclaration(t *testing.T) {
	power := dsl.BindNumber("entering creature power")
	card := dsl.NewCreature("Power watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.AnotherCreatureYouControlEnters().WithPower(power),
			dsl.GainLife(dsl.You, dsl.FromNumber(power)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
