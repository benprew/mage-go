package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControllerLifeAtLeastStartingLifePublicDeclaration(t *testing.T) {
	card := dsl.NewEnchantment("Life watcher", "", dsl.WithAbility(dsl.Static(dsl.While(
		dsl.ControllerLifeAtLeastStartingLife(),
		dsl.ModifyEachPT(dsl.ControlledPermanents(dsl.IsCreature), dsl.Fixed(1), dsl.Fixed(1)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
