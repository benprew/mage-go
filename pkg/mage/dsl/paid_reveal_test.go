package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidRevealPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("revealed card")
	card := dsl.NewInstant("Synthetic", "", dsl.WithAbility(dsl.Spell(
		dsl.IfCardHasType(dsl.FromCard(paid), core.TypeCreature,
			dsl.GainLife(dsl.You, dsl.Fixed(1))))),
		dsl.WithAdditionalCost(dsl.RevealFromHandCostWithCard(
			dsl.IsCreatureCard, "Reveal a creature card", paid)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
