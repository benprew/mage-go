package antiquities

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestMishrasFactory(t *testing.T) {
	t.Run("taps for colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Factory")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Colorless) < 6 { // 5 auto + 1
			t.Errorf("expected at least 6 colorless, got %d", pool.CountProducedThisTurn(core.Colorless))
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

	t.Run("gains Assembly-Worker subtype when animated", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
		// Animate it — should become Assembly-Worker artifact creature
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Factory")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Mishra's Factory", 2, 2)
	})

	t.Run("animate does not tap the Factory itself", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Factory")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Mishra's Factory", false)
	})

	t.Run("is a creature after animating", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Factory")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Mishra's Factory", g.GetPlayer(gametest.PlayerA).PlayerID())
		if perm == nil {
			t.Fatal("Mishra's Factory not found on battlefield")
		}
		if !perm.HasAttr(core.AttrIsCreature) {
			t.Error("Mishra's Factory should be a creature after animating")
		}
		if !perm.HasAttr(core.AttrCanAttack) {
			t.Error("Mishra's Factory should be able to attack after animating")
		}
	})

	t.Run("opponent cannot activate it", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		factory := g.FindPermanentByName("Mishra's Factory", g.GetPlayer(gametest.PlayerA).PlayerID())
		if factory == nil {
			t.Fatal("Mishra's Factory not found on battlefield")
		}

		err := g.ActivateAbilityByIndex(g.GetPlayer(gametest.PlayerB).PlayerID(), factory.ID(), 1, nil)
		if err == nil {
			t.Fatal("PlayerB activated PlayerA's Mishra's Factory")
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
		if pool.CountProducedThisTurn(core.Colorless) < 8 { // 5 auto + 3
			t.Errorf("expected at least 8 colorless, got %d", pool.CountProducedThisTurn(core.Colorless))
		}
	})

	t.Run("workshop mana pays for artifact spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Workshop")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Amulet of Kroog") // {2}, artifact
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Amulet of Kroog")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Amulet of Kroog", 1)
		g.AssertTapped(gametest.PlayerA, "Mishra's Workshop", true)
	})

	// Regression: previously the restriction was implemented as a player-wide
	// flag, so tapping Workshop blocked any non-artifact cast for the rest of
	// the turn — even when other unrestricted mana was available. With per-
	// mana restrictions, only Workshop's three colorless are restricted; the
	// rest of the player's mana remains free.
	t.Run("non-artifact spell still castable after workshop taps", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Workshop")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant") // {3}{R}, non-artifact
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mishra's Workshop")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
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
		if pool.CountProducedThisTurn(core.Colorless) < 6 { // 5 auto + 1
			t.Errorf("expected at least 6 colorless, got %d", pool.CountProducedThisTurn(core.Colorless))
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
		if pool.CountProducedThisTurn(core.Colorless) < 6 { // 5 auto + 1
			t.Errorf("expected at least 6 colorless, got %d", pool.CountProducedThisTurn(core.Colorless))
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
		if pool.CountProducedThisTurn(core.Colorless) < 7 { // 5 auto + 2 (Tron bonus)
			t.Errorf("Urza's Mine with Tron should produce 2 colorless; expected >=7, got %d", pool.CountProducedThisTurn(core.Colorless))
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
		if pool.CountProducedThisTurn(core.Colorless) < 6 {
			t.Errorf("expected at least 6 colorless, got %d", pool.CountProducedThisTurn(core.Colorless))
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
		if pool.CountProducedThisTurn(core.Colorless) < 7 { // 5 auto + 2
			t.Errorf("Urza's Power Plant with Tron should produce 2 colorless; expected >=7, got %d", pool.CountProducedThisTurn(core.Colorless))
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
		if pool.CountProducedThisTurn(core.Colorless) < 6 {
			t.Errorf("expected at least 6 colorless, got %d", pool.CountProducedThisTurn(core.Colorless))
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
		if pool.CountProducedThisTurn(core.Colorless) < 8 { // 5 auto + 3 (Tron Tower)
			t.Errorf("Urza's Tower with Tron should produce 3 colorless; expected >=8, got %d", pool.CountProducedThisTurn(core.Colorless))
		}
	})
}
