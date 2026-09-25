package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPermanentCountComposition(t *testing.T) {
	value := dsl.CountPermanents(dsl.ControlledPermanents(dsl.IsCreature))
	card := dsl.NewInstant("Count", "", dsl.Spell(dsl.GainLife(dsl.You, value)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(value)
	if info.Role != "permanent-count" || len(info.Children) != 1 {
		t.Fatalf("metadata = %+v", info)
	}
}
