package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDividePublicDeclaration(t *testing.T) {
	x := dsl.BindNumber("x")
	card := dsl.NewCreature("Half X", "", 1, 1, dsl.WithAbility(dsl.Activate(
		dsl.Costs(), dsl.Sequence(
			dsl.SnapshotNumber(dsl.Fixed(5), x),
			dsl.DealDamageFrom(dsl.This, dsl.You, dsl.DivideRoundUp(dsl.FromNumber(x), dsl.Fixed(2))),
		))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
