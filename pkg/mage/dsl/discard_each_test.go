package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDiscardEachPublicDeclaration(t *testing.T) {
	card := dsl.NewSorcery("Wheel-like", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.DiscardEach(dsl.EachPlayer),
		dsl.DrawEach(dsl.EachPlayer, dsl.Fixed(7)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
