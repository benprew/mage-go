package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestLifeGainModifierCloneIsolation(t *testing.T) {
	for _, pruneOriginal := range []bool{false, true} {
		name := "prune-clone"
		if pruneOriginal {
			name = "prune-original"
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			original := NewGame(a, b)
			first := NewEnchantment("First", "")
			second := NewEnchantment("Second", "")
			original.PutOnBattlefield(first, a.PlayerID())
			original.PutOnBattlefield(second, a.PlayerID())
			original.AddLifeGainModifier(first.ID(), func(_ *Game, _ uuid.UUID, amount int) int { return amount + 1 })
			original.AddLifeGainModifier(second.ID(), func(game *Game, playerID uuid.UUID, amount int) int {
				if game.GetPlayer(playerID).Life() > 20 {
					return amount * 3
				}
				return amount * 2
			})
			clone := original.Clone()
			pruned, intact := clone, original
			if pruneOriginal {
				pruned, intact = original, clone
			}
			pruned.RemoveFromBattlefield(pruned.FindPermanent(first.ID()))
			assertGain := func(game *Game, want int) {
				t.Helper()
				if got := game.ApplyLifeGainModifiers(a.PlayerID(), 3); got != want {
					t.Fatalf("modified gain = %d, want %d", got, want)
				}
			}
			assertGain(pruned, 6)
			assertGain(intact, 8)
			pruned.AddLifeGainModifier(second.ID(), func(_ *Game, _ uuid.UUID, amount int) int { return amount + 5 })
			assertGain(pruned, 11)
			assertGain(intact, 8)
			grandchild := pruned.Clone()
			grandchild.GetPlayer(a.PlayerID()).GainLife(1)
			assertGain(grandchild, 14)
			assertGain(pruned, 11)
			pruned.RemoveFromBattlefield(pruned.FindPermanent(second.ID()))
			assertGain(pruned, 3)
			assertGain(grandchild, 14)
			assertGain(intact, 8)
		})
	}
}
