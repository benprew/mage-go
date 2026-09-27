package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLifeTotalOfPublicDeclaration(t *testing.T) {
	opponent := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	card := dsl.NewSorcery("Synthetic life total", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(dsl.LoseLife(opponent, dsl.Fixed(3)), dsl.GainLife(dsl.You, dsl.LifeTotalOf(opponent))),
		dsl.WithTargets(opponent),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
