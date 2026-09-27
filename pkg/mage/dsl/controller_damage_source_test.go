package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControllerDamageSourcePublicDeclaration(t *testing.T) {
	amount := dsl.BindNumber("amount")
	source := dsl.BindDamageSource("source")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ControllerIsDealtDamage(amount).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.You, dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
