package legends

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestRelicBarrier(t *testing.T) {
	t.Run("tap to tap target artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Relic Barrier")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Mana Battery")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Relic Barrier", "Black Mana Battery")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Relic Barrier tapped itself (tap cost) and tapped the Black Mana Battery
		g.AssertTapped(gametest.PlayerA, "Relic Barrier", true)
		g.AssertTapped(gametest.PlayerB, "Black Mana Battery", true)
	})
}
