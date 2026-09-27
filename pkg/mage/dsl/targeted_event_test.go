package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestBecomesTargetedPublicDeclaration(t *testing.T) {
	actor := dsl.BindPlayer("actor")
	card := dsl.NewCreature("Watcher", "", 2, 2,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.BecomesTargeted(dsl.This).ByOpponent().WithController(actor),
			dsl.GainLife(dsl.FromPlayer(actor), dsl.Fixed(1)))))
	if card == nil || dsl.InspectDefinition(dsl.BecomesTargeted(dsl.This)).Role != "event-pattern" {
		t.Fatal("targeting pattern is not available")
	}
}
