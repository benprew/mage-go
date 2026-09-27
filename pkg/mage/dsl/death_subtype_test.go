package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCreatureDeathSubtypePublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Goblin watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.AnotherCreatureDies().OfSubtype("Goblin"),
			dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
