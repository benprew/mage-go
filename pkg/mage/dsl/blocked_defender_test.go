package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestBecomesBlockedDefenderPublicDeclaration(t *testing.T) {
	defender := dsl.BindPlayer("defending player")
	card := dsl.NewCreature("Synthetic blocker watcher", "", 2, 2, dsl.WithAbility(dsl.WhenEvent(
		dsl.BecomesBlocked(dsl.This).WithDefendingPlayer(defender),
		dsl.Discard(dsl.FromPlayer(defender), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
