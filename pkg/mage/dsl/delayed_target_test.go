package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDelayedTargetAuthorSurface(t *testing.T) {
	target := dsl.TargetPlayer("future player")
	options := []dsl.DelayedStepOption{dsl.WithTargets(target), dsl.WithInterveningIf(dsl.Controls(dsl.IsArtifact))}
	body := dsl.AtNext(dsl.EachEndStep, dsl.GainLife(target, dsl.Fixed(2)), options...)
	card := dsl.NewSorcery("Schedule", "", dsl.Spell(body))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(body).Role != "delayed-step" {
		t.Fatal("missing delayed descriptor")
	}
}
