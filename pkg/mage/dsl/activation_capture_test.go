package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestActivationCaptureComposition(t *testing.T) {
	n := dsl.BindNumber("credits")
	var capture = dsl.CaptureNumberBeforePayment(dsl.CountersOn(dsl.This, core.Credit), n)
	var _ dsl.ActivationOption = capture
	var _ dsl.GrantedActivationOption = capture
	var _ dsl.ModalActivationOption = capture
	c := dsl.NewArtifact("Capture", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.SacrificeSourceCost()),
		dsl.GainLife(dsl.You, dsl.FromNumber(n)), capture)))
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	descriptor := dsl.InspectDefinition(capture)
	if descriptor.Role != "activation-number-capture" || len(descriptor.Children) != 2 {
		t.Fatalf("descriptor = %+v", descriptor)
	}
}
