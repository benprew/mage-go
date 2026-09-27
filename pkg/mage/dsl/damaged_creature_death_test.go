package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	dsl "github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDamagedCreatureDeathDeclaration(t *testing.T) {
	card := dsl.NewCreature("Synthetic watcher", "", 1, 5,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.CreatureDies().DamagedByThisTurn(),
			dsl.GainLife(dsl.You, dsl.Fixed(1)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
