package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidPermanentAuthorSurface(t *testing.T) {
	paid := dsl.BindPermanent("paid")
	card := dsl.NewInstant("Offering", "", dsl.WithAbility(dsl.Spell(
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(paid)), dsl.You, dsl.Fixed(1)))),
		dsl.WithAdditionalCost(dsl.SacrificeMatchingCostWithPermanent(dsl.IsCreature, "Sacrifice a creature", paid)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
