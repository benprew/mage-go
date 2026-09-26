package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCombatHistoryPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Combat History Watcher", "{2}", 2, 2,
		dsl.WithAbility(dsl.At(dsl.EachEndOfCombat,
			dsl.IfState(dsl.SourceAttackedOrBlockedThisCombatCondition(),
				dsl.GainLife(dsl.You, dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
