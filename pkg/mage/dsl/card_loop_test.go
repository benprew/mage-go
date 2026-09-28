package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestForEachSavedCardAuthorExample(t *testing.T) {
	cards := dsl.BindCards("drawn")
	member := dsl.BindCard("member")
	card := dsl.NewInstant("Synthetic", "", dsl.Spell(dsl.Sequence(
		dsl.Draw(dsl.You, dsl.Fixed(2), dsl.DrawCardsInto(cards)),
		dsl.ForEachSavedCard(cards, member, dsl.RevealCard(dsl.FromCard(member))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, ability := range []mage.Ability{
		dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.Draw(dsl.You, dsl.Fixed(1), dsl.DrawCardsInto(cards)), dsl.ForEachSavedCard(cards, member, dsl.RevealCard(dsl.FromCard(member))))),
		dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Sequence(dsl.Draw(dsl.You, dsl.Fixed(1), dsl.DrawCardsInto(cards)), dsl.ForEachSavedCard(cards, member, dsl.RevealCard(dsl.FromCard(member))))),
	} {
		permanent := dsl.NewArtifact("Synthetic permanent", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(permanent.Name(), permanent); err != nil {
			t.Fatal(err)
		}
	}
}
