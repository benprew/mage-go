package mage

import (
	"strings"
	"testing"
)

func TestLegacyPaymentDefinitionInspection(t *testing.T) {
	paid := DrawFor(You, Fixed(2))
	unpaid := GainLifeFor(You, Fixed(3))
	effect := IfPlayerPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid, unpaid)
	info := InspectDefinition(effect)
	if info.Role != "legacy-payment-branch" || info.Text != "Pay?" || len(info.Children) != 4 {
		t.Fatalf("payment metadata = %+v", info)
	}
	if info.Children[2].Properties.DrawCount != 2 || info.Children[3].Properties.LifeGain != 3 {
		t.Fatalf("branch metadata = %+v", info.Children)
	}
	if info.Properties.DrawCount != 2 || info.Properties.LifeGain != 3 {
		t.Fatalf("summary = %+v", info.Properties)
	}
	unless := InspectDefinition(UnlessTargetPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid))
	if len(unless.Children) != 3 || unless.Children[2].Properties.DrawCount != 2 {
		t.Fatalf("unless metadata = %+v", unless)
	}
}

func TestLegacyPaymentDefinitionValidation(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	saved := BindPermanent("saved")
	var missing *sequenceEffect
	payment := func(paid, unpaid Effect) Effect {
		return IfPlayerPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid, unpaid)
	}
	tests := []struct {
		name    string
		effect  Effect
		path    string
		message string
	}{
		{"paid target", payment(Destroy(target), nil), ".paid.subject", "target binding requires"},
		{"unpaid target", UnlessTargetPays(SelectController(), ManaCostOf("{1}"), "Pay?", Destroy(target)), ".unpaid.subject", "target binding requires"},
		{"paid saved read", payment(Destroy(From(saved)), nil), ".paid.subject", "producer before this read"},
		{"unpaid saved read", payment(nil, Destroy(From(saved))), ".unpaid.subject", "producer before this read"},
		{"sibling assignment", payment(Snapshot(This, saved), Destroy(From(saved))), ".unpaid.subject", "producer before this read"},
		{"escaped assignment", CompositeEffects("sequence", payment(Snapshot(This, saved), nil), Destroy(From(saved))), ".effects[1].subject", "producer before this read"},
		{"nil payer", IfPlayerPays(nil, ManaCostOf("{1}"), "Pay?", nil, nil), ".payer", "non-nil expression"},
		{"nil cost", IfPlayerPays(SelectController(), nil, "Pay?", nil, nil), ".cost", "non-nil expression"},
		{"typed nil paid", payment(missing, nil), ".paid", "non-nil expression"},
		{"typed nil unpaid", payment(nil, missing), ".unpaid", "non-nil expression"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card := NewArtifact("Payment", "", WithAbility(NewActivatedAbility(test.effect, ManaCostOf("{0}"))))
			err := ValidateCardDefinition("Payment", card)
			if err == nil || !strings.Contains(err.Error(), "Payment: abilities[0].") || !strings.Contains(err.Error(), test.path) || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("diagnostic = %v, want path %s and %s", err, test.path, test.message)
			}
		})
	}
}

func TestLegacyPaymentDefinitionValidScopes(t *testing.T) {
	saved := BindPermanent("saved")
	payment := func(paid, unpaid Effect) Effect {
		return IfPlayerPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid, unpaid)
	}
	for _, effect := range []Effect{
		payment(nil, nil),
		payment(Sequence(Snapshot(This, saved), Destroy(From(saved))), nil),
		payment(nil, Sequence(Snapshot(This, saved), Destroy(From(saved)))),
		CompositeEffects("outer snapshot", Snapshot(This, saved), payment(Destroy(From(saved)), Destroy(From(saved)))),
	} {
		card := NewArtifact("Payment", "", WithAbility(NewActivatedAbility(effect, ManaCostOf("{0}"))))
		if err := ValidateCardDefinition("Payment", card); err != nil {
			t.Fatal(err)
		}
	}
	target := DeclarePermanentTarget("creature", IsCreature)
	card := NewInstant("Payment", "", WithAbility(Spell(payment(Destroy(target), nil), WithTargets(target))))
	if err := ValidateCardDefinition("Payment", card); err != nil {
		t.Fatal(err)
	}
	bad := NewInstant("Payment", "", WithAbility(Spell(payment(Sacrifice(This), nil))))
	if err := ValidateCardDefinition("Payment", bad); err == nil || !strings.Contains(err.Error(), ".paid.subject: battlefield source") {
		t.Fatalf("spell source diagnostic = %v", err)
	}
}
