package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPublicOptionalRevealDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid")
	revealed := dsl.BindCard("revealed")
	card := dsl.NewInstant("Optional reveal", "",
		dsl.WithAbility(dsl.Spell(dsl.Sequence(
			dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
			dsl.IfCardHasType(dsl.FromCard(revealed), core.TypeCreature,
				dsl.GainLife(dsl.You, dsl.Fixed(2)))))),
		dsl.WithAdditionalCost(dsl.OptionalRevealFromHandCost(
			dsl.IsCreatureCard, "Reveal a creature card", "Reveal a creature?", paid, revealed)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
