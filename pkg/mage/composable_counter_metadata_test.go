package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableCounterMetadata(t *testing.T) {
	for _, tc := range []struct {
		name    string
		counter core.CounterType
		amount  ValueSource
		outcome Outcome
	}{
		{"positive", core.P1P1, Fixed(2), OutcomeBenefit},
		{"negative", core.M1M1, Fixed(2), OutcomeDetriment},
		{"zero", core.P1P1, Fixed(0), OutcomeUnknown},
		{"less than zero", core.P1P1, Fixed(-1), OutcomeUnknown},
		{"dynamic", core.P1P1, XValue(), OutcomeUnknown},
		{"other", core.Loyalty, Fixed(2), OutcomeUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := DeclarePermanentTarget("creature", IsCreature)
			for _, effect := range []ResolvingEffect{AddCountersTo(target, tc.counter, counterTestAmount(tc.amount)), AddCounters(tc.counter, tc.amount)} {
				if got := effect.Properties().Outcome; got != tc.outcome {
					t.Fatalf("outcome = %v, want %v", got, tc.outcome)
				}
			}
		})
	}
}

func counterTestAmount(value ValueSource) ResolutionValue {
	if fixed, ok := value.(fixedValue); ok {
		return fixed
	}
	return ValueAtResolution(value)
}
