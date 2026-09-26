package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidCardPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("paid creature")
	spell := dsl.NewInstant("Offering", "",
		dsl.WithAbility(dsl.Spell(
			dsl.IfCardHasType(dsl.FromCard(paid), core.TypeCreature,
				dsl.GainLife(dsl.You, dsl.Fixed(2))))),
		dsl.WithAdditionalCost(dsl.SacrificeMatchingCostWithCard(
			dsl.IsCreature, "Sacrifice a creature", paid)))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}
