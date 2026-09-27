package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPlayerWithUniqueHighestLifePublicDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Synthetic life selector", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(),
		dsl.GainLife(dsl.PlayerWithUniqueHighestLife(), dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
