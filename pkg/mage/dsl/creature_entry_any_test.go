package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnotherCreatureEntersPublicDeclaration(t *testing.T) {
	power := dsl.BindNumber("entry power")
	card := dsl.NewCreature("Watcher", "", 6, 6, dsl.WithAbility(dsl.WhenEvent(
		dsl.AnotherCreatureEnters().WithPower(power),
		dsl.AddCounters(dsl.This, core.P1P1, dsl.FromNumber(power)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
