package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOwnerOfPublicDeclaration(t *testing.T) {
	target := dsl.TargetPermanent("permanent", dsl.AnyPermanent)
	owner := dsl.BindPlayer("owner")
	card := dsl.NewInstant("Owner", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(
			dsl.SnapshotPlayer(dsl.OwnerOf(target), owner),
			dsl.GainLife(dsl.FromPlayer(owner), dsl.Fixed(1)),
		),
		dsl.WithTargets(target),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
