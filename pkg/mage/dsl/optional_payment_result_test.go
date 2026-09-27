package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOptionalPaymentResultPublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.OptionalCostInto(dsl.LifePayCost(2), "Pay two life?", paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
