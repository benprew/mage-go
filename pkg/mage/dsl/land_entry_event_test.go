package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
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

func TestLandEntrySubtypeDSL(t *testing.T) {
	controller := dsl.BindPlayer("controller")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandEnters().OfSubtype("Swamp").WithController(controller),
		dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.Fixed(3)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(dsl.LandEnters().OfSubtype("Swamp")); got.Role != "event-pattern" {
		t.Fatalf("inspection: %+v", got)
	}
}

func TestLandEntryActorDSL(t *testing.T) {
	actor := dsl.BindPlayer("actor")
	controller := dsl.BindPlayer("controller")
	pattern := dsl.LandEnters().OfSubtype("Swamp").WithActor(actor).WithController(controller)
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern,
		dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(actor), dsl.Fixed(3)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(pattern); got.Role != "event-pattern" {
		t.Fatalf("inspection: %+v", got)
	}
}
