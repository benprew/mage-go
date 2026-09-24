package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDrawStepReplacementComposition(t *testing.T) {
	var rule = dsl.SkipDrawStepForLife(2)
	card := dsl.NewEnchantment("Static skip", "", dsl.WithAbility(dsl.Static(rule)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(card.Abilities()[0])
	if len(info.Children) != 1 || info.Children[0].Role != "static-draw-step-replacement" {
		t.Fatalf("missing replacement definition: %+v", info)
	}
}
