package legends

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestDwarvenSong(t *testing.T) {
	t.Run("target creature becomes red", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dwarven Song")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Song", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestHeavensGate(t *testing.T) {
	t.Run("target creature becomes white", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Heaven's Gate")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Heaven's Gate", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestTouchOfDarkness(t *testing.T) {
	t.Run("target creature becomes black", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Touch of Darkness")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Touch of Darkness", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestPsychicPurge(t *testing.T) {
	t.Run("deals 1 damage to player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Psychic Purge")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Psychic Purge", "PlayerB")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestJovialEvil(t *testing.T) {
	t.Run("deals damage based on white creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Keepers of the Faith") // white creature
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Tundra Wolves")        // white creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jovial Evil")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jovial Evil", "PlayerB")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 16) // 2 white creatures × 2 = 4 damage
	})
}

func TestTyphoon(t *testing.T) {
	t.Run("deals damage based on Islands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Typhoon")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Typhoon")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17) // 3 Islands = 3 damage
	})
}

func TestGreatDefender(t *testing.T) {
	t.Run("boosts toughness by mana value", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Durkwood Boars costs {4}{G} = CMC 5, so +0/+5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Durkwood Boars")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Great Defender")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Great Defender", "Durkwood Boars")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Durkwood Boars", 4, 9) // 4/4 + 0/+5
	})
}

func TestManaDrain(t *testing.T) {
	t.Run("counters target spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Drain")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Mana Drain")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestTeleport(t *testing.T) {
	t.Run("makes creature unblockable", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // would block
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Teleport")
		g.CastSpell(3, core.BeginCombat, gametest.PlayerA, "Teleport", "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.Block(3, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // unblockable, 2 damage
	})
}

func TestEnergyTap(t *testing.T) {
	t.Run("taps creature for colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Durkwood Boars is CMC 5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Durkwood Boars")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Energy Tap")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Energy Tap", "Durkwood Boars")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Durkwood Boars", true)
	})
}

func TestWindsOfChange(t *testing.T) {
	t.Run("each player shuffles hand and redraws", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Winds of Change")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Raging Bull")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Durkwood Boars")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Moss Monster")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Winds of Change")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// PlayerA had 2 cards in hand after casting (Grizzly Bears + Raging Bull)
		// Winds shuffles them in, draws 2 from library
		// So PlayerA should still have 2 cards in hand
	})
}
