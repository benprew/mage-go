package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLifeLostIntoPublicDeclaration(t *testing.T) {
	lost := dsl.BindNumber("life lost")
	body := dsl.Sequence(
		dsl.LoseLifeEach(dsl.OpponentsOf(dsl.You), dsl.Fixed(4), dsl.LifeLostInto(lost)),
		dsl.GainLife(dsl.You, dsl.FromNumber(lost)),
	)
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}
