package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSacrificePermanentCapturePublicDeclaration(t *testing.T) {
	paid := dsl.BindPermanent("paid creature")
	card := dsl.NewCreature("Sacrifice watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.CreatureYouSacrifice().WithPermanent(paid),
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(paid)), dsl.You, dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
