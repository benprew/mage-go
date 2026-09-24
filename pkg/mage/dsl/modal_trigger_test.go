package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedModalTriggerComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	modes := []dsl.ModeDefinition{
		dsl.ModeOf("Return", dsl.ReturnToHand(target), dsl.WithTargets(target)),
		dsl.ModeOf("Draw", dsl.Sequence(dsl.Draw(dsl.You, dsl.Fixed(1)), dsl.GainLife(dsl.You, dsl.Fixed(1)))),
	}
	for _, tc := range []struct {
		name, role string
		ability    mage.TriggeredAbility
	}{
		{"Entry", "event-trigger", dsl.WhenEventModes(dsl.Enters(dsl.This), modes...)},
		{"Death", "event-trigger", dsl.WhenEventModes(dsl.Dies(dsl.This), modes...)},
		{"Upkeep", "step-trigger", dsl.AtModes(dsl.YourUpkeep, modes...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := dsl.NewCreature(tc.name, "", 1, 1, dsl.WithAbility(tc.ability))
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			info := dsl.InspectDefinition(tc.ability)
			if info.Role != tc.role || len(info.Children) != 3 || info.Children[1].Role != "mode" || info.Children[1].Text != "Return" || info.Children[2].Text != "Draw" {
				t.Fatalf("metadata = %+v", info)
			}
		})
	}
}
