package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaProductionsForAbility(t *testing.T) {
	t.Run("dedicated mana ability", func(t *testing.T) {
		got := ManaProductionsForAbility(NewMultiManaAbility(
			ManaProduction{Color: Colorless, Amount: 2},
		))
		if len(got) != 1 || got[0].Color != Colorless || got[0].Amount != 2 {
			t.Fatalf("unexpected productions: %+v", got)
		}
	})

	t.Run("tap activated mana ability", func(t *testing.T) {
		ability := NewActivatedAbility(AddMana(Green, 1), Tap())
		got := ManaProductionsForAbility(ability)
		if len(got) != 1 || got[0].Color != Green || got[0].Amount != 1 {
			t.Fatalf("unexpected productions: %+v", got)
		}
	})

	t.Run("non-mana ability", func(t *testing.T) {
		ability := NewActivatedAbility(DrawCards(Fixed(1)), Tap())
		if got := ManaProductionsForAbility(ability); got != nil {
			t.Fatalf("unexpected productions: %+v", got)
		}
	})
}
