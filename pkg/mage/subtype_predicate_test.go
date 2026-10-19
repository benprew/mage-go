package mage

import "testing"

func TestPermanentSubtypeTargetFamily(t *testing.T) {
	for _, tc := range []struct {
		family SubtypeFamily
		name   string
		valid  bool
	}{
		{SubtypeCreature, "Wall", true}, {SubtypeCreature, "Assembly-Worker", true},
		{SubtypeLand, "Island", true}, {SubtypeLand, "Power-Plant", true}, {SubtypeEnchantment, "Aura", true},
		{SubtypeCreature, "Island", false}, {SubtypeLand, "Wall", false}, {SubtypeEnchantment, "Wall", false},
		{SubtypeLand, "", false}, {SubtypeCreature, " Wall", false}, {SubtypeFamily(99), "Wall", false},
	} {
		if err := ValidateSubtypePredicate(tc.family, tc.name); (err == nil) != tc.valid {
			t.Fatalf("family=%v subtype=%s error=%v", tc.family, tc.name, err)
		}
	}
}
