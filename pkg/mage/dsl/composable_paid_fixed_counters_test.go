package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestFixedCounterPaymentPublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid")
	card := dsl.NewArtifact("Battery", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.RemoveCountersCostInto(core.Charge, 2, paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
