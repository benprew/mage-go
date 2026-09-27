package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControllerGainedLifeThisTurnPublicDeclaration(t *testing.T) {
	card := dsl.NewInstant("Conditional", "", dsl.WithAbility(dsl.Spell(dsl.IfState(
		dsl.ControllerGainedLifeThisTurn(), dsl.Draw(dsl.You, dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
