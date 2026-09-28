package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestBlockersOfPublicDeclaration(t *testing.T) {
	attacker := dsl.TargetPermanent("attacker", dsl.IsCreature)
	card := dsl.NewInstant("Count blockers", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.BlockersOf(attacker))),
		dsl.WithTargets(attacker))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
