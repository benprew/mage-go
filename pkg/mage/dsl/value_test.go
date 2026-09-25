package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedPermanentValueComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	body := dsl.Sequence(dsl.Snapshot(target, saved), dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.PowerOf(dsl.From(saved)), dsl.ToughnessOf(target))))
	card := dsl.NewInstant("Values", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(dsl.PowerOf(target))
	if metadata.Role != "permanent-value" || len(metadata.Children) != 1 {
		t.Fatalf("metadata = %+v", metadata)
	}
	var fixed dsl.ResolutionValue = dsl.Fixed(2)
	var legacy dsl.ValueSource = dsl.Fixed(2)
	if fixed.Text() != dsl.ValueAtResolution(legacy).Text() {
		t.Fatal("legacy adapter lost text")
	}
}

func TestTypedArithmeticComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	value := dsl.MinValue(dsl.Fixed(0), dsl.MaxValue(dsl.Fixed(-4), dsl.SubtractValues(dsl.Fixed(1), dsl.ToughnessOf(target))))
	body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.Fixed(4), value))
	card := dsl.NewInstant("Arithmetic", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(value)
	if metadata.Role != "arithmetic-value" || metadata.Text != "minimum" || len(metadata.Children) != 2 {
		t.Fatalf("metadata = %+v", metadata)
	}
	if value.Text() != "minimum(0, maximum(-4, subtract(1, toughness of subject)))" {
		t.Fatalf("text = %q", value.Text())
	}
}

func TestTypedMultiplyComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	value := dsl.MultiplyValues(dsl.Fixed(2), dsl.ValueAtResolution(dsl.XValue()))
	card := dsl.NewSorcery("Multiply", "{X}{U}", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.Tap(target), dsl.AddCounters(target, core.Stun, value)), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(value)
	if metadata.Role != "arithmetic-value" || metadata.Text != "multiply" || len(metadata.Children) != 2 {
		t.Fatalf("metadata = %+v", metadata)
	}
	if got := dsl.MultiplyValues(dsl.Fixed(2), dsl.Fixed(3)).Text(); got != "multiply(2, 3)" {
		t.Fatalf("text = %q", got)
	}
}
