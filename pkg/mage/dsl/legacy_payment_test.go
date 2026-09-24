package dsl_test

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLegacyPaymentTypedChildren(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	for _, body := range []dsl.ResolvingEffect{
		dsl.IfPlayerPays(dsl.SelectController(), dsl.ManaCostOf("{1}"), "Pay?", dsl.Destroy(target), dsl.Draw(dsl.You, dsl.Fixed(1))),
		dsl.UnlessTargetPays(dsl.SelectController(), dsl.ManaCostOf("{1}"), "Pay?", dsl.Destroy(target)),
	} {
		info := dsl.InspectDefinition(body)
		if info.Role != "legacy-payment-branch" || len(info.Children) < 3 || !info.Children[2].Properties.Destroys {
			t.Fatalf("payment inspection = %+v", info)
		}
		valid := dsl.NewInstant("Payment", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target))))
		if err := mage.ValidateCardDefinition("Payment", valid); err != nil {
			t.Fatal(err)
		}
		invalid := dsl.NewInstant("Payment", "", dsl.WithAbility(dsl.Spell(body)))
		if err := mage.ValidateCardDefinition("Payment", invalid); err == nil || !strings.Contains(err.Error(), "target binding requires") {
			t.Fatalf("target diagnostic = %v", err)
		}
	}
}
