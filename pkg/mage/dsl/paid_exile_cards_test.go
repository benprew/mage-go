package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidExileCardsPublicDeclaration(t *testing.T) {
	paid := dsl.BindCards("exiled cards")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.ExileFromGraveyardCostWithCards(2, paid)),
		dsl.GainLife(dsl.You, dsl.CountCards(dsl.FromCards(paid))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
