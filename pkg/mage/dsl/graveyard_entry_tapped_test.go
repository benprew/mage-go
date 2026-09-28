package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestReturnCardToBattlefieldTappedAuthorExample(t *testing.T) {
	discarded := dsl.BindCards("discarded")
	picked := dsl.BindCard("picked")
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Synthetic return", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.Discard(player, dsl.Fixed(2), dsl.DiscardCardsInto(discarded)),
		dsl.ChooseUpToOneCard(dsl.You, dsl.FromCards(discarded), dsl.IsLandCard, picked),
		dsl.ReturnCardToBattlefield(dsl.FromCard(picked), dsl.You, dsl.EnterTapped()),
	), dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
