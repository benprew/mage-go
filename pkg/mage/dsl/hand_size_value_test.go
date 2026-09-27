package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestHandSizeOfPublicDeclaration(t *testing.T) {
	opponent := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	card := dsl.NewSorcery("Synthetic hand spell", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(
			dsl.DiscardEntireHand(dsl.You),
			dsl.Draw(dsl.You, dsl.HandSizeOf(opponent)),
		),
		dsl.WithTargets(opponent),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
