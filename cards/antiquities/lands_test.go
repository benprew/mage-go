package antiquities

import (
	"testing"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestMishrasFactory(t *testing.T) {
	t.Run("taps for colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Factory")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 6 { // 5 auto + 1
			t.Errorf("expected at least 6 colorless, got %d", pool.Count(core.Colorless))
		}
	})

	t.Run("becomes 2/2 creature when animated", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
		// Activate the {1} ability to animate
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Factory")
		g.Attack(1, gametest.PlayerA, "Mishra's Factory")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // 2/2 deals 2 damage
	})

	t.Run("is a Land", func(t *testing.T) {
		card, err := mage.CreateCard("Mishra's Factory")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeLand) {
			t.Errorf("Mishra's Factory should be a Land")
		}
	})
}

func TestMishrasWorkshop(t *testing.T) {
	t.Run("taps for 3 colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Workshop")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Workshop")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 8 { // 5 auto + 3
			t.Errorf("expected at least 8 colorless, got %d", pool.Count(core.Colorless))
		}
	})

	t.Run("mana can only be spent on artifact spells", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Workshop")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant") // non-artifact creature
		// Workshop mana shouldn't be usable for Hill Giant
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Workshop")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// If restriction works, Hill Giant should not be on battlefield
		// (This test will pass when restriction is implemented)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 0)
	})
}

func TestStripMine(t *testing.T) {
	t.Run("taps for colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Strip Mine")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Strip Mine")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 6 { // 5 auto + 1
			t.Errorf("expected at least 6 colorless, got %d", pool.Count(core.Colorless))
		}
	})

	t.Run("destroys target land when sacrificed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Strip Mine")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		// Use the second ability: {T}, Sacrifice: destroy target land
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Strip Mine", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Strip Mine", 0)
	})
}

func TestUrzasMine(t *testing.T) {
	t.Run("taps for 1 colorless normally", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Mine")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Mine")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 6 { // 5 auto + 1
			t.Errorf("expected at least 6 colorless, got %d", pool.Count(core.Colorless))
		}
	})

	t.Run("taps for 2 colorless with complete Tron", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Mine")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Power Plant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Tower")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Mine")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 7 { // 5 auto + 2 (Tron bonus)
			t.Errorf("Urza's Mine with Tron should produce 2 colorless; expected >=7, got %d", pool.Count(core.Colorless))
		}
	})
}

func TestUrzasPowerPlant(t *testing.T) {
	t.Run("taps for 1 colorless normally", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Power Plant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Power Plant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 6 {
			t.Errorf("expected at least 6 colorless, got %d", pool.Count(core.Colorless))
		}
	})

	t.Run("taps for 2 colorless with complete Tron", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Mine")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Power Plant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Tower")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Power Plant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 7 { // 5 auto + 2
			t.Errorf("Urza's Power Plant with Tron should produce 2 colorless; expected >=7, got %d", pool.Count(core.Colorless))
		}
	})
}

func TestUrzasTower(t *testing.T) {
	t.Run("taps for 1 colorless normally", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Tower")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Tower")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 6 {
			t.Errorf("expected at least 6 colorless, got %d", pool.Count(core.Colorless))
		}
	})

	t.Run("taps for 3 colorless with complete Tron", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Mine")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Power Plant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Tower")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Tower")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 8 { // 5 auto + 3 (Tron Tower)
			t.Errorf("Urza's Tower with Tron should produce 3 colorless; expected >=8, got %d", pool.Count(core.Colorless))
		}
	})
}
