package heuristic

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPrecombatEnchantmentCreatureAbilities(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options []mage.CardOption
		want    bool
	}{
		{name: "no precombat ability"},
		{name: "haste", options: []mage.CardOption{mage.WithKeyword(core.Haste)}, want: true},
		{name: "first strike", options: []mage.CardOption{mage.WithKeyword(core.FirstStrike)}},
		{name: "spell damage", options: []mage.CardOption{mage.WithAbility(mage.NewSpell(mage.DealDamage(mage.Fixed(1))))}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := append([]mage.CardOption{mage.WithCardType(core.TypeEnchantment)}, tc.options...)
			card := mage.NewCreature(tc.name, "{R}", 1, 1, options...)
			if got := isPrecombatSpell(card); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	card := mage.NewCreature("Hasty Enchantment", "{R}", 1, 1, mage.WithCardType(core.TypeEnchantment), mage.WithKeyword(core.Haste))
	found := false
	if allocations := testing.AllocsPerRun(100, func() { found = isPrecombatSpell(card) }); allocations != 0 {
		t.Fatalf("precombat keyword check allocated %g times", allocations)
	}
	if !found {
		t.Fatal("precombat check did not find haste")
	}
}
