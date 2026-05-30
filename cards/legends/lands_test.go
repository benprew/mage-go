package legends

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
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

func TestHammerheim(t *testing.T) {
	t.Run("taps for red mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hammerheim")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Hammerheim", 1)
	})

	t.Run("removes landwalk from creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hammerheim")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Lost Soul") // has swampwalk
		// Use Hammerheim to remove landwalk
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Hammerheim", "Lost Soul")
		// Lost Soul attacks — but without swampwalk it can be blocked
		g.Attack(2, gametest.PlayerB, "Lost Soul")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.Block(2, gametest.PlayerA, "Grizzly Bears", "Lost Soul")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Lost Soul was blocked — player A takes no damage
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestTheTabernacleAtPendrellVale(t *testing.T) {
	t.Run("destroys creatures that cannot pay upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Tabernacle at Pendrell Vale")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// PlayerB has no lands to pay {1} during upkeep
		// Turn 2 is PlayerB's turn — upkeep triggers
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// Bears should be destroyed (couldn't pay {1})
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("creatures survive if controller can pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Tabernacle at Pendrell Vale")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		// PlayerB has a Forest to pay {1}
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// Bears survive because controller could pay {1}
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestTolaria(t *testing.T) {
	t.Run("taps for blue mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tolaria")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Tolaria", 1)
	})

	t.Run("removes banding from creature during upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tolaria")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus") // 1/1 banding flying
		// Activate during PlayerB's upkeep (turn 2)
		g.ActivateAbility(2, core.Upkeep, gametest.PlayerA, "Tolaria", "Mesa Pegasus")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// Mesa Pegasus should have lost banding
		g.AssertHasAbility(gametest.PlayerB, "Mesa Pegasus", core.Banding, false)
	})
}

func TestUrborg(t *testing.T) {
	t.Run("taps for black mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urborg")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Urborg", 1)
	})

	t.Run("removes first strike from creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urborg")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Tundra Wolves") // 1/1 first strike
		// Remove first strike from Tundra Wolves
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Urborg", "Tundra Wolves")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Tundra Wolves should not have first strike
		g.AssertHasAbility(gametest.PlayerB, "Tundra Wolves", core.FirstStrike, false)
	})
}

func TestAdventurersGuildhouse(t *testing.T) {
	t.Run("grants banding to green legendary creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Adventurers' Guildhouse")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jasmine Boreal") // {3}{G}{W} legendary creature
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Jasmine Boreal is green+white legendary — should have banding
		g.AssertHasAbility(gametest.PlayerA, "Jasmine Boreal", core.Banding, true)
	})
	t.Run("does not grant banding to non-legendary creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Adventurers' Guildhouse")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green, not legendary
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Banding, false)
	})
}

func TestCathedralOfSerra(t *testing.T) {
	t.Run("grants banding to white legendary creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cathedral of Serra")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sir Shandlar of Eberyn") // {3}{G}{W}{W} legendary
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Sir Shandlar of Eberyn", core.Banding, true)
	})
}

func TestMountainStronghold(t *testing.T) {
	t.Run("grants banding to red legendary creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain Stronghold")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ramirez DePietro") // {3}{U}{B} legendary — NOT red
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tor Wauki")        // {2}{B}{B}{R} legendary — has red
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Ramirez DePietro", core.Banding, false)
		g.AssertHasAbility(gametest.PlayerA, "Tor Wauki", core.Banding, true)
	})
}

func TestUnholyCitadel(t *testing.T) {
	t.Run("grants banding to black legendary creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Unholy Citadel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol'kanar the Swamp King") // {2}{U}{B}{R} legendary — has black
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Sol'kanar the Swamp King", core.Banding, true)
	})
}
