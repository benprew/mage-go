package core

import "testing"

func TestHuskCounter(t *testing.T) {
	if got := Husk.String(); got != "Husk" {
		t.Errorf("Husk.String() = %q, want Husk", got)
	}
	if Husk >= NumCounters {
		t.Errorf("Husk = %d must be below NumCounters = %d", Husk, NumCounters)
	}
	if Husk.PowerBoost() != 0 || Husk.ToughnessBoost() != 0 {
		t.Error("husk counters must not modify power or toughness")
	}
}

func TestTheDarkCounters(t *testing.T) {
	cases := []struct {
		counter CounterType
		name    string
		power   int
		tough   int
	}{
		{P0P1, "+0/+1", 0, 1},
		{P2P0, "+2/+0", 2, 0},
		{P0P2, "+0/+2", 0, 2},
		{Hunger, "Hunger", 0, 0},
	}
	for _, tc := range cases {
		if got := tc.counter.String(); got != tc.name {
			t.Errorf("%v.String() = %q, want %q", tc.counter, got, tc.name)
		}
		if tc.counter >= NumCounters {
			t.Errorf("%v = %d must be below NumCounters = %d", tc.counter, tc.counter, NumCounters)
		}
		if gotP := tc.counter.PowerBoost(); gotP != tc.power {
			t.Errorf("%v.PowerBoost() = %d, want %d", tc.counter, gotP, tc.power)
		}
		if gotT := tc.counter.ToughnessBoost(); gotT != tc.tough {
			t.Errorf("%v.ToughnessBoost() = %d, want %d", tc.counter, gotT, tc.tough)
		}
	}
}
