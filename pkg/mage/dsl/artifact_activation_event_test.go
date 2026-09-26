package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestArtifactActivationEventDSL(t *testing.T) {
	controller := dsl.BindPlayer("controller")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ArtifactAbilityActivatedWithoutTapCost().WithController(controller),
		dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))
	if card == nil || dsl.InspectDefinition(dsl.ArtifactAbilityActivatedWithoutTapCost()).Role != "event-pattern" {
		t.Fatal("artifact activation pattern is not available")
	}
}
