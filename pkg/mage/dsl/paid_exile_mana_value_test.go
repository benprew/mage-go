package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidGraveyardExileManaValueDeclaration(t *testing.T) {
	paid := dsl.BindNumber("exiled mana value")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.ExileMatchingCardFromGraveyardCostInto(dsl.IsCreatureCard, "Exile a creature card", paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
