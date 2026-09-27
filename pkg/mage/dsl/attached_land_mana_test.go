package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAttachedLandTappedForManaComposition(t *testing.T) {
	player := dsl.BindPlayer("producer")
	amount := dsl.BindNumber("produced")
	card := dsl.NewAura("Land watcher", "", dsl.WithCastTarget(dsl.TargetLand()), dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().OfAttachedLand().OfManaType(core.Blue).WithPlayer(player).WithAmount(amount),
		dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
