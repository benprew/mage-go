package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestReceivesDamageSourcePublicDeclaration(t *testing.T) {
	amount := dsl.BindNumber("amount")
	source := dsl.BindDamageSource("source")
	card := dsl.NewCreature("Synthetic Damage Watcher", "", 1, 5, dsl.WithAbility(dsl.WhenEvent(
		dsl.ReceivesDamage(dsl.This, amount).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.You, dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
