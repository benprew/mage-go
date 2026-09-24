package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDelayedModesAuthorSurface(t *testing.T) {
	target := dsl.TargetPlayer("future target")
	options := []dsl.ModalDelayedStepOption{
		dsl.ModeOf("Life", dsl.GainLife(target, dsl.Fixed(2)), dsl.WithTargets(target)),
		dsl.ModeOf("Other", dsl.GainLife(dsl.You, dsl.Fixed(1))),
		dsl.WithInterveningIf(dsl.Controls(dsl.IsArtifact)),
	}
	body := dsl.AtNextModes(dsl.EachEndStep, options...)
	card := dsl.NewSorcery("Schedule", "", dsl.Spell(body))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(body).Role != "delayed-step" || body.Properties().Outcome != mage.OutcomeBenefit {
		t.Fatal("missing metadata")
	}
}
