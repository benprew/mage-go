package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDrawCardsResultDeclaration(t *testing.T) {
	drawn := dsl.BindCards("drawn")
	card := dsl.NewInstant("Synthetic draw", "", dsl.Spell(dsl.Sequence(
		dsl.Draw(dsl.You, dsl.Fixed(2), dsl.DrawCardsInto(drawn)),
		dsl.GainLife(dsl.You, dsl.CountCards(dsl.FromCards(drawn))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
