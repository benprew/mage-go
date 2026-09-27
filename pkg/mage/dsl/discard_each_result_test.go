package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDiscardEachTotalPublicDeclaration(t *testing.T) {
	total := dsl.BindNumber("discarded")
	card := dsl.NewSorcery("Shared discard total", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.DiscardEach(dsl.EachPlayer, dsl.DiscardTotalInto(total)),
		dsl.GainLife(dsl.You, dsl.FromNumber(total)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
