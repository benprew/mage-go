package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestGraveyardLeaveEventPublicDeclaration(t *testing.T) {
	count := dsl.BindNumber("left")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.OneOrMoreCardsLeaveYourGraveyard().WithCount(count),
		dsl.GainLife(dsl.You, dsl.FromNumber(count)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
