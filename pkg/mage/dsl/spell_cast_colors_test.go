package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellCastColorsSpentDSL(t *testing.T) {
	colors := dsl.BindNumber("colors")
	pattern := dsl.SpellCast().OfType(core.TypeInstant).WithColorsSpent(colors)
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern,
		dsl.GainLife(dsl.You, dsl.FromNumber(colors)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(pattern); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
}
