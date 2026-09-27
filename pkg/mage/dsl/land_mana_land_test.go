package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLandManaLandPublicDeclaration(t *testing.T) {
	land := dsl.BindPermanent("producing land")
	player := dsl.BindPlayer("activator")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().WithLand(land).WithPlayer(player),
		dsl.Tap(dsl.From(land)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
