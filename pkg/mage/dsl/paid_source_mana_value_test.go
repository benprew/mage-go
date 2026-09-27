package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidSourceManaValuePublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid mana value")
	card := dsl.NewCreature("Synthetic", "{2}{U}", 1, 1,
		dsl.WithAbility(dsl.Activate(
			dsl.Costs(dsl.SacrificeMatchingIncludingSourceCostWithManaValue(
				dsl.And(dsl.IsCreature, dsl.HasColorFilter(dsl.Blue)), "Sacrifice a blue creature", paid)),
			dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
