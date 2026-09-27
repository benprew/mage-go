package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOpponentDamageHistoryDeclaration(t *testing.T) {
	card := dsl.NewCreature("History watcher", "", 1, 1,
		dsl.WithAbility(dsl.At(dsl.EachEndStep,
			dsl.GainLife(dsl.You, dsl.Fixed(1)),
			dsl.WithInterveningIf(dsl.SourceDamagedOpponentThisTurnCondition()))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
