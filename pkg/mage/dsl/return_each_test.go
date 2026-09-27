package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestReturnEachToHandPublicDeclaration(t *testing.T) {
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Return owned artifacts", "{1}{U}", dsl.WithAbility(dsl.Spell(
		dsl.ReturnEachToHand(dsl.PermanentsOwnedBy(player, dsl.IsArtifact)),
		dsl.WithTargets(player),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
