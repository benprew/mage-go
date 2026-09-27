package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDiscardAmountIntoPublicDeclaration(t *testing.T) {
	discarded := dsl.BindNumber("discarded")
	body := dsl.Sequence(
		dsl.DiscardEntireHand(dsl.You, dsl.DiscardAmountInto(discarded)),
		dsl.Draw(dsl.You, dsl.FromNumber(discarded)),
	)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}
