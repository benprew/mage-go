package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCounterRemovalResultPublicDeclaration(t *testing.T) {
	removed := dsl.BindNumber("removed counters")
	target := dsl.TargetPermanent("permanent", dsl.AnyPermanent)
	card := dsl.NewInstant("Synthetic removal", "", dsl.Spell(dsl.Sequence(
		dsl.RemoveCountersFromWithResult(target, dsl.Charge, dsl.Fixed(3), removed),
		dsl.GainLife(dsl.You, dsl.FromNumber(removed)),
	), dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
