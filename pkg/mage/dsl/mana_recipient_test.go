package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAddManaToPlayerDeclaration(t *testing.T) {
	producer := dsl.BindPlayer("producer")
	card := dsl.NewEnchantment("Mana watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().WithPlayer(producer),
		dsl.AddManaToPlayer(dsl.FromPlayer(producer), core.Green, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
