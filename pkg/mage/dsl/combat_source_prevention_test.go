package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCombatSourcePreventionPublicDeclaration(t *testing.T) {
	blocker := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewInstant("Combat prevention", "", dsl.WithAbility(dsl.Spell(
		dsl.IfPermanentBlocking(blocker, dsl.Sequence(
			dsl.PreventCombatDamageBy(blocker),
			dsl.PreventCombatDamageByEach(dsl.BlockedAttackersOf(blocker)))),
		dsl.WithTargets(blocker))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
