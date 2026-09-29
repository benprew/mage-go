package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCombatStateBranchPublicDefinition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewInstant("Combat check", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.IfPermanentAttacking(target, dsl.GainLife(dsl.You, dsl.Fixed(1))),
		dsl.IfPermanentBlocking(target, dsl.GainLife(dsl.You, dsl.Fixed(1))),
	), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
