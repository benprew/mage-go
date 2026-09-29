package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestBlockedAttackersOfPublicDeclaration(t *testing.T) {
	blocker := dsl.TargetPermanent("blocker", dsl.IsCreature)
	card := dsl.NewInstant("Count blocked attackers", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.BlockedAttackersOf(blocker))),
		dsl.WithTargets(blocker))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
