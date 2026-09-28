package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidSourcePermanentPublicDeclaration(t *testing.T) {
	paid := dsl.BindPermanent("paid")
	card := dsl.NewCreature("Test", "", 2, 2, dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.SacrificeMatchingIncludingSourceCostWithPermanent(dsl.IsCreature, "Sacrifice a creature", paid)),
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(paid)), dsl.You, dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
