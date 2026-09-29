package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDistinctDiscardPublicDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.DiscardCardsWithDifferentNamesCost(3)),
		dsl.Draw(dsl.You, dsl.Fixed(5)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	node := dsl.InspectDefinition(dsl.DiscardCardsWithDifferentNamesCost(3))
	if node.Role != "discard-cost" || node.Text != "Discard 3 cards with different names" {
		t.Fatalf("inspection = %+v", node)
	}
}
