package limited

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for cards registered in alpha_enchantments.go.

func TestInvisibility(t *testing.T) {
	t.Run("creature_unblockable_except_walls", func(t *testing.T) {
		// Invisibility: Enchanted creature can't be blocked except by Walls.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3 non-Wall
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Invisibility")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Invisibility", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant is not a Wall, so it can't block invisible creature.
		// Bears should deal 2 damage unblocked.
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("walls_can_still_block", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone") // 0/8 Wall
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Invisibility")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Invisibility", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Wall of Stone IS a Wall, so it can block invisible creature.
		// No damage to PlayerB.
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestLure(t *testing.T) {
	t.Run("all_creatures_must_block", func(t *testing.T) {
		// Lure: All creatures able to block enchanted creature do so.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions") // 2/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Gray Ogre")      // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lure")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lure", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
		// Even though PlayerB tries to block Hill Giant, Lure forces blocks on Bears.
		g.Block(1, gametest.PlayerB, "Savannah Lions", "Hill Giant")
		g.Block(1, gametest.PlayerB, "Gray Ogre", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// All of PlayerB's creatures must block the Lured creature (Grizzly Bears),
		// not Hill Giant. So Hill Giant gets through unblocked for 3 damage.
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestWildGrowth(t *testing.T) {
	t.Run("enchanted_land_produces_extra_G", func(t *testing.T) {
		// Wild Growth: Whenever enchanted land is tapped for mana, its controller
		// adds an additional {G}.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wild Growth")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wild Growth", "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Auto-mana adds {G} to pay for Wild Growth. Forest taps for {G},
		// Wild Growth adds another {G}. Total: 1 (auto) + 1 (Forest) + 1 (bonus) = 3G.
		// Without bonus: 1 + 1 = 2G.
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Green) < 3 {
			t.Errorf("Wild Growth should add extra {G} when Forest taps; expected >= 3 green, got %d", pool.CountProducedThisTurn(core.Green))
		}
	})
}

func TestCircleOfProtection(t *testing.T) {
	t.Run("prevents_red_damage", func(t *testing.T) {
		// Circle of Protection: Red -- {1}: The next time a red source of your
		// choice would deal damage to you this turn, prevent that damage.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Circle of Protection: Red")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt") // red source
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.ActivateInResponseTo(gametest.PlayerB, "Circle of Protection: Red")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// CoP:Red should prevent the 3 red damage.
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("doesnt_prevent_other_colors", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Circle of Protection: Red")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Terror")                  // black spell
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")       // non-black creature for Terror
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hypnotic Specter") // black 2/2 flyer
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Circle of Protection: Red")
		g.Attack(1, gametest.PlayerA, "Hypnotic Specter")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hypnotic Specter is black, not red. CoP:Red shouldn't prevent its damage.
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestColorWards(t *testing.T) {
	t.Run("grants_protection_from_color", func(t *testing.T) {
		// Black Ward: Enchanted creature has protection from black.
		// (Testing one representative of the cycle.)
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Ward")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror") // black spell targeting non-black creature
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Ward", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Terror", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Protection from black: can't be targeted by Terror (black).
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("ward_not_removed_by_protection", func(t *testing.T) {
		// Special rule: Wards grant protection but don't fall off due to
		// protection from their own color (they have a special exception).
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Ward")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Ward", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Black Ward should still be attached even though it grants protection from black.
		g.AssertPermanentCount(gametest.PlayerA, "Black Ward", 1)
		g.AssertAttachedTo(gametest.PlayerA, "Black Ward", "Grizzly Bears")
	})
}

func TestLaceCycle(t *testing.T) {
	t.Run("changes_permanent_color", func(t *testing.T) {
		// Deathlace: Target spell or permanent becomes black.
		// (Testing one representative of the cycle.)
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Deathlace")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Deathlace", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should now be black.
		perm := g.FindPermanentByName("Grizzly Bears", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Grizzly Bears not found")
		}
		hasBlack := false
		for _, col := range perm.Colors() {
			if col == core.Black {
				hasBlack = true
			}
		}
		if !hasBlack {
			t.Errorf("Deathlace should make Grizzly Bears black")
		}
	})
}
