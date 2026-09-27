package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidXCountersPublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("removed")
	card := dsl.NewArtifact("Counter source", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.RemoveXCountersFromSourceCostInto(core.Charge, paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
