package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidExileCardPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("exiled card")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.ExileMatchingCardFromGraveyardCostWithCard(
			dsl.IsCreatureCard, "Exile a creature card", paid)),
		dsl.IfCardHasType(dsl.FromCard(paid), dsl.TypeCreature,
			dsl.GainLife(dsl.You, dsl.Fixed(1))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
