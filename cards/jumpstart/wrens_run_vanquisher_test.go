package jumpstart

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestWrensRunVanquisher_RevealOrPay(t *testing.T) {
	t.Run("reveal Elf branch when an Elf is in hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wren's Run Vanquisher")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Llanowar Elves")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wren's Run Vanquisher")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Wren's Run Vanquisher", 1)
		g.AssertHandCount(gametest.PlayerA, "Llanowar Elves", 1)
	})
}
