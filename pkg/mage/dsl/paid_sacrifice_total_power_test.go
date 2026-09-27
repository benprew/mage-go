package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidSacrificeTotalPowerPublicDeclaration(t *testing.T) {
	total := dsl.BindNumber("total")
	card := dsl.NewArtifact("Offering", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.SacrificeNMatchingCostWithTotalPower(2, dsl.IsCreature, "Sacrifice two creatures", total)),
		dsl.GainLife(dsl.You, dsl.FromNumber(total)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
