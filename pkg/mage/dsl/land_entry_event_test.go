package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLandEntryEventDSL(t *testing.T) {
	controller := dsl.BindPlayer("land controller")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandEnters().WithController(controller),
		dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.Fixed(2)))))
	if card == nil || dsl.InspectDefinition(dsl.LandEnters()).Role != "event-pattern" {
		t.Fatal("land entry pattern is not available")
	}
}
