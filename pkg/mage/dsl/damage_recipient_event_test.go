package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDamagePermanentRecipientPublicDeclaration(t *testing.T) {
	recipient := dsl.BindPermanent("recipient")
	card := dsl.NewCreature("Damage watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.DealsDamage(dsl.This, dsl.BindNumber("dealt")).WithPermanent(recipient),
			dsl.AddCounters(dsl.From(recipient), dsl.P1P1, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
