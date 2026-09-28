package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDiscardCardResultPublicDeclaration(t *testing.T) {
	card := dsl.BindCard("discarded")
	player := dsl.TargetPlayer("player")
	spell := dsl.NewInstant("Synthetic", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(
			dsl.Discard(player, dsl.Fixed(1), dsl.DiscardCardInto(card)),
			dsl.ReturnCardToHand(dsl.FromCard(card))),
		dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}
