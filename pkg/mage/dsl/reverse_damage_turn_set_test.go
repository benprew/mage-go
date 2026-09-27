package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestReverseTurnDamageSetDeclaration(t *testing.T) {
	card := dsl.NewCreature("Watcher", "", 1, 4,
		dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This),
			dsl.TapEach(dsl.PermanentsThatDealtDamageToThisTurn(dsl.This,
				dsl.MatchingPermanents(dsl.IsCreature))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
