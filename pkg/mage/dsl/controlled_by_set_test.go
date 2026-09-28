package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControlledByPermanentSetDeclaration(t *testing.T) {
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Controlled artifacts", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.PermanentsControlledBy(player, dsl.IsArtifact))),
		dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
