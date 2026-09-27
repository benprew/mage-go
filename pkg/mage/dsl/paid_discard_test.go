package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidDiscardPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("discarded card")
	card := dsl.NewArtifact("Discard capture", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.DiscardCardCostInto(paid)),
		dsl.IfCardHasType(dsl.FromCard(paid), core.TypeLand,
			dsl.GainLife(dsl.You, dsl.Fixed(2))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
