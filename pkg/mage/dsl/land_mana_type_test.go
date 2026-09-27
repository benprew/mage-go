package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLandManaTypePublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("activator")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().OfManaType(core.Blue).WithPlayer(player),
		dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
