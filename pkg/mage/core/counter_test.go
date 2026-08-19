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
