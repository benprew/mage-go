package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDeathPermanentPublicDeclaration(t *testing.T) {
	dead := dsl.BindPermanent("departing creature")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.AnotherCreatureDies().WithPermanent(dead),
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(dead)), dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
