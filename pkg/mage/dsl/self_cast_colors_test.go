package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSelfCastColorsDSL(t *testing.T) {
	card := dsl.NewInstant("Converge", "", dsl.Spell(
		dsl.Draw(dsl.You, dsl.ColorsSpentToCastThisSpell())))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
