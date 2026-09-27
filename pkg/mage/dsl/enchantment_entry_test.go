package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestEnchantmentEntryDeclaration(t *testing.T) {
	entered := dsl.BindPermanent("entered enchantment")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.EnchantmentYouControlEnters().WithPermanent(entered),
		dsl.AddCounters(dsl.From(entered), core.P1P1, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
