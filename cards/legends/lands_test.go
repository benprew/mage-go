package legends

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestPendelhaven(t *testing.T) {
	t.Run("taps for green mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pendelhaven")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Pendelhaven is a legendary land
		g.AssertPermanentCount(gametest.PlayerA, "Pendelhaven", 1)
	})

	t.Run("boosts 1/1 creature +1/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pendelhaven")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pit Scorpion") // 1/1
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Pendelhaven", "Pit Scorpion")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Pit Scorpion was 1/1, now 2/3
		g.AssertPowerToughness(gametest.PlayerA, "Pit Scorpion", 2, 3)
	})
}

func TestKarakas(t *testing.T) {
	t.Run("bounces legendary creature to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Karakas")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol'kanar the Swamp King")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Karakas", "Sol'kanar the Swamp King")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Sol'kanar bounced to PlayerB's hand
		g.AssertPermanentCount(gametest.PlayerB, "Sol'kanar the Swamp King", 0)
		g.AssertHandCount(gametest.PlayerB, "Sol'kanar the Swamp King", 1)
	})
}
