package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestUntapResultPublicDeclaration(t *testing.T) {
	untapped := dsl.BindNumber("untapped")
	target := dsl.TargetPermanent("target", dsl.IsCreature)
	card := dsl.NewInstant("Synthetic untap", "", dsl.Spell(dsl.Sequence(
		dsl.UntapPermanentWithResult(target, dsl.UntappedThisWayInto(untapped)),
		dsl.IfPositive(dsl.FromNumber(untapped), dsl.GainLife(dsl.You, dsl.Fixed(1))),
	), dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
