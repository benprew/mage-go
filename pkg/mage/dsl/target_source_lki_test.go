package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTargetSourceLKIReferenceDeclaration(t *testing.T) {
	recipient := dsl.TargetDamageRecipient("recipient")
	card := dsl.NewCreature("Synthetic Departed Source", "{G}", 2, 2,
		dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.SacrificeSourceCost()),
			dsl.Sequence(dsl.DealDamageFrom(dsl.This, recipient, dsl.Fixed(2)),
				dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(recipient))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
