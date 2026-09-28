package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControlledSinceTurnStartDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Gate", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(), dsl.GainLife(dsl.You, dsl.Fixed(1)),
		dsl.ActivationIf(dsl.ControlledSinceTurnStart(dsl.This)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
