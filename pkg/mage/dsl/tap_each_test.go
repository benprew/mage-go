package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTapEachAuthorExample(t *testing.T) {
	card := dsl.NewInstant("Blue tap", "", dsl.WithAbility(dsl.Spell(
		dsl.TapEach(dsl.MatchingPermanents(dsl.HasColorFilter(dsl.Blue))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
