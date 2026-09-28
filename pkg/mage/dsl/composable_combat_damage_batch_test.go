package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCombatDamageBatchAuthorSurface(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	var card dsl.Card = dsl.NewEnchantment("Combat Batch Watcher", "{G}",
		dsl.WithAbility(dsl.WhenEvent(
			dsl.OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(player),
			dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestCombatDamageBatchSubtypeAuthorSurface(t *testing.T) {
	var card dsl.Card = dsl.NewEnchantment("Non-Human Watcher", "{G}",
		dsl.WithAbility(dsl.WhenEvent(
			dsl.OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype("Human"),
			dsl.Draw(dsl.You, dsl.Fixed(1)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
