package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPermanentCaptureComposition(t *testing.T) {
	host := dsl.BindPermanent("host")
	capture := dsl.CapturePermanentBeforePayment(dsl.AttachedPermanent, host)
	var _ dsl.ActivationOption = capture
	var _ dsl.GrantedActivationOption = capture
	var _ dsl.ModalActivationOption = capture
	c := dsl.NewAura("Capture", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.SacrificeSourceCost()), dsl.Regenerate(dsl.From(host)), capture)))
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	descriptor := dsl.InspectDefinition(capture)
	if descriptor.Role != "activation-permanent-capture" || len(descriptor.Children) != 2 {
		t.Fatalf("descriptor = %+v", descriptor)
	}
}
