package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSingularLifeLossResultPublicDeclaration(t *testing.T) {
	lost := dsl.BindNumber("life lost")
	target := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	body := dsl.Sequence(
		dsl.LoseLifeForWithResult(target, dsl.Fixed(4), dsl.LifeLostInto(lost)),
		dsl.GainLife(dsl.You, dsl.FromNumber(lost)),
	)
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.CreatureDies(), body, dsl.WithTargets(target)))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}
