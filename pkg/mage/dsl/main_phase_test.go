package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestMainPhaseComposition(t *testing.T) {
	for _, pattern := range []dsl.TurnStepPattern{dsl.YourFirstMainPhase, dsl.YourPostcombatMainPhase} {
		body := dsl.GainLife(dsl.You, dsl.Fixed(1))
		for _, ability := range []mage.TriggeredAbility{
			dsl.At(pattern, body),
			dsl.AtModes(pattern, dsl.ModeOf("One", body), dsl.ModeOf("Two", body)),
			dsl.At(dsl.YourUpkeep, dsl.AtNext(pattern, body)),
		} {
			card := dsl.NewEnchantment("Main phase", "", dsl.WithAbility(ability))
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		if info := dsl.InspectDefinition(pattern); info.Role != "turn-step-pattern" || info.Text == "" {
			t.Fatalf("inspection = %+v", info)
		}
	}
}
