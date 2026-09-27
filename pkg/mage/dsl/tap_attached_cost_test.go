package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTapAttachedPermanentCostDeclaration(t *testing.T) {
	card := dsl.NewAura("Synthetic gate", "{1}{B}{B}",
		dsl.WithCastTarget(dsl.TargetLand()),
		dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapAttachedPermanentCost()),
			dsl.GainLife(dsl.You, dsl.Fixed(1)),
			dsl.ActivationIf(dsl.PermanentIsUntapped(dsl.AttachedPermanent)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
