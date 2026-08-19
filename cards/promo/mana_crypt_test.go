package promo

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestManaCrypt_UpkeepCoinFlip(t *testing.T) {
	t.Run("lost flip deals 3 damage to controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Crypt")
		g.SetCoinFlipResults([]bool{false})
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 17)
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("won flip deals no damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Crypt")
		g.SetCoinFlipResults([]bool{true})
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("does not trigger during opponent upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mana Crypt")
		g.SetCoinFlipResults([]bool{false})
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestManaCrypt_AddsTwoColorlessMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Crypt")
	g.SetCoinFlipResults([]bool{true})
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mana Crypt")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Mana Crypt", true)
	// Activated-ability scripts seed 10 colorless mana before adding Mana Crypt's two.
	g.AssertManaProduced(gametest.PlayerA, core.Colorless, 12)
}
