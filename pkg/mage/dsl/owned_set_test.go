package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOwnedPermanentSetDeclaration(t *testing.T) {
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Owned artifacts", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.PermanentsOwnedBy(player, dsl.IsArtifact))),
		dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
