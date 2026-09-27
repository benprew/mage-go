package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSelfEntryColorsSpentDSL(t *testing.T) {
	colors := dsl.BindNumber("entry colors")
	card := dsl.NewCreature("Entry colors", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.Enters(dsl.This).WithColorsSpent(colors),
		dsl.GainLife(dsl.You, dsl.FromNumber(colors)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
