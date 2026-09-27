package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestMillCostPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Upkeep", "", 1, 1,
		dsl.WithAbility(dsl.At(dsl.YourUpkeep,
			dsl.UnlessPaid(dsl.You, dsl.Costs(dsl.MillCost(2)), dsl.Sacrifice(dsl.This)))))
	if err := mage.ValidateCardDefinition("Upkeep", card); err != nil {
		t.Fatal(err)
	}
}
