package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPaidTapPermanentDeclaration(t *testing.T) {
	paid := dsl.BindPermanent("paid Merfolk")
	card := dsl.NewArtifact("Test", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.TapNOtherMatchingCostWithPermanent(dsl.IsCreature,
			"Tap another creature", paid)),
		dsl.Untap(dsl.From(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
