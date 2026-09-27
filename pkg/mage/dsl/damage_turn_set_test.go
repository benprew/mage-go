package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTurnDamagePermanentSetDeclaration(t *testing.T) {
	card := dsl.NewCreature("Turn history", "", 1, 5,
		dsl.WithAbility(dsl.At(dsl.YourUpkeep,
			dsl.TapEach(dsl.PermanentsDealtDamageByThisTurn(dsl.This,
				dsl.MatchingPermanents(dsl.IsCreature))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
