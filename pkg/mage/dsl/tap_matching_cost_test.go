package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTapNOtherMatchingCostDeclaration(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	ability := dsl.Activate(
		dsl.Costs(dsl.TapCost(dsl.This), dsl.TapNOtherMatchingCost(3,
			dsl.And(dsl.IsCreature, dsl.HasColorFilter(core.White)),
			"Tap three untapped white creatures you control")),
		dsl.Destroy(target), dsl.WithTargets(target))
	card := dsl.NewCreature("Synthetic", "{5}{W}", 2, 6, dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
