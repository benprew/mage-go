package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLifeGainedThisTurnValuePublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Synthetic life history", "", 1, 1,
		dsl.WithAbility(dsl.At(dsl.YourEndStep,
			dsl.GainLife(dsl.You, dsl.LifeGainedThisTurn(dsl.You)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
