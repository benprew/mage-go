package core

import "testing"

func TestManaValueIncludesColorless(t *testing.T) {
	cost := ManaCost{Generic: 2, HasX: true, XCount: 1, Colorless: 2, Blue: 1}
	if got := cost.CMC(); got != 5 {
		t.Fatalf("mana value = %d, want 5", got)
	}
}

func TestColorlessManaCostRoundTrip(t *testing.T) {
	cost := ParseManaCost("{C}")
	if cost.Colorless != 1 || cost.IsZero() || cost.CMC() != 1 || cost.String() != "{C}" {
		t.Fatalf("incorrect colorless cost: %+v", cost)
	}
	mixed := ParseManaCost("{X}{2}{C}{U}{C}")
	if mixed.Colorless != 2 || mixed.Blue != 1 || mixed.CMC() != 5 || mixed.XCount != 1 {
		t.Fatalf("incorrect mixed cost: %+v", mixed)
	}
	if got := ParseManaCost(mixed.String()); got.Colorless != mixed.Colorless || got.CMC() != mixed.CMC() {
		t.Fatalf("formatting lost the colorless requirements: %s", mixed)
	}
}
