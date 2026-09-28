package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDiscardCardsResultPublicDeclaration(t *testing.T) {
	discarded := dsl.BindCards("discarded")
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Synthetic", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.Discard(player, dsl.Fixed(2), dsl.DiscardCardsInto(discarded)),
		dsl.GainLife(dsl.You, dsl.CountCards(dsl.FromCards(discarded))),
	), dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
