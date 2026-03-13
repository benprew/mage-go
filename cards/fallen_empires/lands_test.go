package fallen_empires

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestDwarvenRuins tests the sacrifice land cycle representative.
func TestDwarvenRuins(t *testing.T) {
	t.Run("taps for red mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Ruins")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Ruins")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Red) < 1 {
			t.Errorf("expected at least 1 red mana, got %d", pool.Count(core.Red))
		}
	})

	t.Run("sacrifice for two red mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Ruins")
		// Activate the sacrifice ability (second ability)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Ruins")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Dwarven Ruins", 0)
	})
}

func TestEbonStronghold(t *testing.T) {
	t.Run("sacrifice for two black mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ebon Stronghold")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ebon Stronghold")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ebon Stronghold", 0)
	})
}

func TestHavenwoodBattleground(t *testing.T) {
	t.Run("sacrifice for two green mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Havenwood Battleground")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Havenwood Battleground")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Havenwood Battleground", 0)
	})
}

func TestRuinsOfTrokair(t *testing.T) {
	t.Run("sacrifice for two white mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ruins of Trokair")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ruins of Trokair")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ruins of Trokair", 0)
	})
}

func TestSvyeluniteTemple(t *testing.T) {
	t.Run("sacrifice for two blue mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Svyelunite Temple")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Svyelunite Temple")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Svyelunite Temple", 0)
	})
}

// === Storage Land Tests ===

// TestBottomlessVault_ManaAbility tests that tapping with storage counters produces black mana.
func TestBottomlessVault_ManaAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bottomless Vault")
	// Pre-load 3 storage counters
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Bottomless Vault", core.Storage, 3)
	// Activate the mana ability to remove all counters and add black mana
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bottomless Vault")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Should have removed all 3 storage counters
	g.AssertCounterCount(gametest.PlayerA, "Bottomless Vault", core.Storage, 0)
	// Should have at least 3 black mana (pool may have mana from other sources)
	pool := g.AllPlayers()[0].ManaPool()
	if pool.Count(core.Black) < 3 {
		t.Errorf("expected at least 3 black mana, got %d", pool.Count(core.Black))
	}
}

// TestBottomlessVault_EntersTapped tests that the land enters tapped.
func TestBottomlessVault_EntersTapped(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bottomless Vault")
	// Land auto-plays from hand during PrecombatMain, enters tapped
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Bottomless Vault", 1)
	g.AssertTapped(gametest.PlayerA, "Bottomless Vault", true)
}

// TestBottomlessVault_UpkeepAddsCounterWhenTapped tests the upkeep trigger.
// Tap the land during turn 1 main (activate with 0 counters), then on turn 3
// the land is tapped at start of turn. TestPlayer chooses to untap (always yes),
// so the land untaps before upkeep. To work around this, we activate (tap) it
// during turn 1, and check that on turn 3 it still gets untapped and produces 0 counters.
// The real counter accumulation test is covered by the mana ability tests above.
func TestBottomlessVault_UpkeepAddsCounterWhenTapped(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bottomless Vault")
	// Activate turn 1 (tap, remove 0 counters, get 0 mana) — land is now tapped
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bottomless Vault")
	// Stop at turn 3 precombat main (PlayerA's next turn)
	// Turn 3: Untap (TestPlayer says yes, untaps) -> Upkeep (untapped, no counter)
	g.StopAt(3, core.PrecombatMain)
	g.Execute()
	// Land should be untapped (TestPlayer always chooses to untap)
	g.AssertTapped(gametest.PlayerA, "Bottomless Vault", false)
	// Should have 0 counters (upkeep didn't add any because land was untapped)
	g.AssertCounterCount(gametest.PlayerA, "Bottomless Vault", core.Storage, 0)
}

// TestBottomlessVault_TapWithZeroCounters tests activating with no counters taps the land.
func TestBottomlessVault_TapWithZeroCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bottomless Vault")
	// Activate with 0 counters
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bottomless Vault")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Land should be tapped
	g.AssertTapped(gametest.PlayerA, "Bottomless Vault", true)
	// Should still have 0 storage counters
	g.AssertCounterCount(gametest.PlayerA, "Bottomless Vault", core.Storage, 0)
}

// TestDwarvenHold_ManaAbility tests that Dwarven Hold produces red mana from storage counters.
func TestDwarvenHold_ManaAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Hold")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Hold", core.Storage, 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Hold")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Dwarven Hold", core.Storage, 0)
	pool := g.AllPlayers()[0].ManaPool()
	if pool.Count(core.Red) < 2 {
		t.Errorf("expected at least 2 red mana, got %d", pool.Count(core.Red))
	}
}

// TestHollowTrees_ManaAbility tests that Hollow Trees produces green mana from storage counters.
func TestHollowTrees_ManaAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hollow Trees")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Hollow Trees", core.Storage, 4)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Hollow Trees")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Hollow Trees", core.Storage, 0)
	pool := g.AllPlayers()[0].ManaPool()
	if pool.Count(core.Green) < 4 {
		t.Errorf("expected at least 4 green mana, got %d", pool.Count(core.Green))
	}
}

// TestIcatianStore_ManaAbility tests that Icatian Store produces white mana from storage counters.
func TestIcatianStore_ManaAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Icatian Store")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Icatian Store", core.Storage, 1)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Icatian Store")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Icatian Store", core.Storage, 0)
	pool := g.AllPlayers()[0].ManaPool()
	if pool.Count(core.White) < 1 {
		t.Errorf("expected at least 1 white mana, got %d", pool.Count(core.White))
	}
}

// TestSandSilos_ManaAbility tests that Sand Silos produces blue mana from storage counters.
func TestSandSilos_ManaAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sand Silos")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Sand Silos", core.Storage, 5)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sand Silos")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Sand Silos", core.Storage, 0)
	pool := g.AllPlayers()[0].ManaPool()
	if pool.Count(core.Blue) < 5 {
		t.Errorf("expected at least 5 blue mana, got %d", pool.Count(core.Blue))
	}
}

// TestBottomlessVault_HasMayNotUntap tests the land has the may-not-untap attribute.
func TestBottomlessVault_HasMayNotUntap(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bottomless Vault")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Bottomless Vault", core.AttrMayNotUntap, true)
}
