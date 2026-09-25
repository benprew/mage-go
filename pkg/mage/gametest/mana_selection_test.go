package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationSelectionPrefersNonMana(t *testing.T) {
	for _, dynamic := range []bool{false, true} {
		t.Run(map[bool]string{false: "fixed", true: "dynamic"}[dynamic], func(t *testing.T) {
			tg := NewTestGame(t)
			player := tg.GetPlayer(PlayerA).PlayerID()
			var amount mage.ResolutionValue = mage.Fixed(1)
			if dynamic {
				amount = mage.PowerOf(mage.This)
			}
			land := mage.NewLand("Selection land",
				mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.AddManaAmount(core.Green, amount))),
				mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.GainLifeFor(mage.You, mage.Fixed(2)))),
			)
			perm := tg.PutOnBattlefield(land, player)
			_, index, err := tg.findActivatableAbilityByName(player, land.Name(), nil)
			if err != nil || index != 1 {
				t.Fatalf("selected index %d, error %v; want non-mana index 1", index, err)
			}
			perm.RuntimeAbilities = perm.RuntimeAbilities[:1]
			_, index, err = tg.findActivatableAbilityByName(player, land.Name(), nil)
			if err != nil || index != 0 {
				t.Fatalf("selected index %d, error %v; want mana fallback index 0", index, err)
			}
			perm.Tapped = true
			if _, _, err := tg.findActivatableAbilityByName(player, land.Name(), nil); err == nil {
				t.Fatal("selected an unpaid tap ability")
			}
		})
	}
}
