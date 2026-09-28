package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestExileResultPublicDeclaration(t *testing.T) {
	result := dsl.BindNumber("exiled")
	target := dsl.TargetPermanent("target", dsl.IsCreature)
	card := dsl.NewInstant("Synthetic exile", "", dsl.Spell(dsl.Sequence(
		dsl.ExileWithResult(target, dsl.ExiledThisWayInto(result)),
		dsl.IfPositive(dsl.FromNumber(result), dsl.GainLife(dsl.You, dsl.Fixed(1))),
	), dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
