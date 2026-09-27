package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControllerDrawCardCaptureDeclaration(t *testing.T) {
	drawer, drawn := dsl.BindPlayer("drawer"), dsl.BindCard("drawn")
	card := dsl.NewEnchantment("Draw Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ControllerDrawsCardInto(drawer, drawn),
		dsl.IfCardHasType(dsl.FromCard(drawn), core.TypeLand, dsl.GainLife(dsl.FromPlayer(drawer), dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
