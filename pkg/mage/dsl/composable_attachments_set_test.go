package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAttachedAurasPublicDeclaration(t *testing.T) {
	bonus := dsl.MultiplyValues(dsl.Fixed(2), dsl.CountPermanents(dsl.AttachedAuras(dsl.This)))
	card := dsl.NewCreature("Synthetic wombat", "", 0, 1,
		dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature),
			dsl.ModifyPT(dsl.This, bonus, bonus)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
