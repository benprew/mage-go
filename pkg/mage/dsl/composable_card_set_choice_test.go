package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPublicChooseUpToOneCard(t *testing.T) {
	discarded := dsl.BindCards("discarded")
	chosen := dsl.BindCard("chosen")
	player := dsl.TargetPlayer("player")
	card := dsl.NewSorcery("Choice", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.Discard(player, dsl.Fixed(2), dsl.DiscardCardsInto(discarded)),
		dsl.ChooseUpToOneCard(dsl.You, dsl.FromCards(discarded), dsl.IsLandCard, chosen),
		dsl.ReturnCardToHand(dsl.FromCard(chosen)),
	), dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
