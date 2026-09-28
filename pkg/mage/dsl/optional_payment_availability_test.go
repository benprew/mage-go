package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPerDrawnCardOptionalSacrificeAuthorExample(t *testing.T) {
	cards, member := dsl.BindCards("drawn"), dsl.BindCard("member")
	card := dsl.NewInstant("Synthetic draw", "{X}{U}", dsl.Spell(dsl.Sequence(
		dsl.Draw(dsl.You, dsl.Fixed(2), dsl.DrawCardsInto(cards)),
		dsl.ForEachSavedCard(cards, member, dsl.UnlessPaid(dsl.You,
			dsl.Costs(dsl.SacrificeMatchingIncludingSourceCost(dsl.AnyPermanent, "Sacrifice a permanent")),
			dsl.Discard(dsl.You, dsl.Fixed(1)))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
