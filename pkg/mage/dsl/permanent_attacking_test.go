package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPermanentAttackingPublicDefinition(t *testing.T) {
	card := dsl.NewCreature("Attacking guardian", "", 2, 2,
		dsl.WithAbility(dsl.Static(dsl.While(
			dsl.PermanentIsAttacking(dsl.This),
			dsl.ModifyPT(dsl.This, dsl.Fixed(0), dsl.Fixed(2)),
		))),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
