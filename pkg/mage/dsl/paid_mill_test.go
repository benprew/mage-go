package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidMillCountDeclaration(t *testing.T) {
	paid := dsl.BindNumber("milled")
	card := dsl.NewArtifact("Mill source", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.MillCostInto(2, paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
