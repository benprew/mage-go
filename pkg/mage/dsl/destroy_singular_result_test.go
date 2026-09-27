package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDestroySingularResultPublicDeclaration(t *testing.T) {
	result := dsl.BindNumber("destroyed")
	card := dsl.NewCreature("Test", "", 2, 2, dsl.WithAbility(dsl.At(dsl.YourUpkeep, dsl.Sequence(
		dsl.DestroyWithResult(dsl.This, dsl.DestroyedThisWayInto(result)),
		dsl.IfPositive(dsl.FromNumber(result), dsl.DealDamageFrom(dsl.This, dsl.You, dsl.Fixed(7))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
