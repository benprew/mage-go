package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDiscardCostBatchAuthorDeclaration(t *testing.T) {
	ability := dsl.Activate(
		dsl.Costs(dsl.DiscardCost(2)),
		dsl.Draw(dsl.You, dsl.Fixed(1)),
	)
	card := dsl.NewArtifact("Synthetic discard payment", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
