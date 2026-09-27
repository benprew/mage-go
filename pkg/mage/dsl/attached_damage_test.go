package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAttachedDamagePublicDeclaration(t *testing.T) {
	dealt := dsl.BindNumber("damage dealt")
	card := dsl.NewAura("Linked Aura", "{W}", dsl.WithAbility(
		dsl.WhenEvent(dsl.DealsDamage(dsl.AttachedPermanent, dealt),
			dsl.GainLife(dsl.You, dsl.FromNumber(dealt))),
	))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
