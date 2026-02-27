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

func TestHornOfDeafening(t *testing.T) {
	t.Run("prevents combat damage from target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Horn of Deafening")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4
		// Prevent Craw Wurm's damage on turn 2
		g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Horn of Deafening", "Craw Wurm")
		g.Attack(2, gametest.PlayerB, "Craw Wurm")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Craw Wurm's combat damage prevented
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestArenaOfTheAncients(t *testing.T) {
	t.Run("taps legendary creatures on ETB", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol'kanar the Swamp King")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Arena of the Ancients")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Arena of the Ancients")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Sol'kanar is legendary — should be tapped
		g.AssertTapped(gametest.PlayerA, "Sol'kanar the Swamp King", true)
		// Grizzly Bears is not legendary — should not be tapped
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", false)
	})

	t.Run("legendary creatures do not untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arena of the Ancients")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol'kanar the Swamp King")
		// Sol'kanar attacks on turn 1, becomes tapped
		g.Attack(1, gametest.PlayerA, "Sol'kanar the Swamp King")
		// On turn 3 (PlayerA's next turn), Sol'kanar should still be tapped
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Sol'kanar the Swamp King", true)
	})
}
