package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControllerHandSizeIsPublicDeclaration(t *testing.T) {
	card := dsl.NewLand("Hand gate", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.TapCost(dsl.This)),
		dsl.Draw(dsl.You, dsl.Fixed(1)),
		dsl.ActivationIf(dsl.ControllerHandSizeIs(7)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
