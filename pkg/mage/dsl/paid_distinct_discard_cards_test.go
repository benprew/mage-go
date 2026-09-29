package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidDistinctDiscardCardsPublicDeclaration(t *testing.T) {
	paid := dsl.BindCards("discarded cards")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.DiscardCardsWithDifferentNamesCostWithCards(2, paid)),
		dsl.GainLife(dsl.You, dsl.CountCards(dsl.FromCards(paid))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	node := dsl.InspectDefinition(dsl.DiscardCardsWithDifferentNamesCostWithCards(2, paid))
	if node.Role != "discard-cost" || len(node.Children) != 1 {
		t.Fatalf("inspection = %+v", node)
	}
}
