package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestGainLifeEachPublicDeclaration(t *testing.T) {
	card := dsl.NewSorcery("Shared gain", "", dsl.WithAbility(dsl.Spell(dsl.GainLifeEach(dsl.EachPlayer, dsl.Fixed(3)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
