package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDiscardCardEventPublicDeclaration(t *testing.T) {
	discarder := dsl.BindPlayer("discarder")
	discarded := dsl.BindCard("discarded")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.OpponentDiscardsCard(discarder).WithCard(discarded),
		dsl.IfCardHasType(dsl.FromCard(discarded), core.TypeLand,
			dsl.LoseLife(dsl.FromPlayer(discarder), dsl.Fixed(2))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
