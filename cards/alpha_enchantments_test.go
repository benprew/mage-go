package cards

import (
	"testing"

	"github.com/mage/mage"
)

// Tests for cards registered in alpha_enchantments.go.

func TestInvisibility(t *testing.T) {
	t.Run("creature_unblockable_except_walls", func(t *testing.T) {
		// Invisibility: Enchanted creature can't be blocked except by Walls.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")    // 3/3 non-Wall
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Invisibility")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Invisibility", "Grizzly Bears")
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		g.Block(1, mage.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Hill Giant is not a Wall, so it can't block invisible creature.
		// Bears should deal 2 damage unblocked.
		g.AssertLife(mage.PlayerB, 18)
	})

	t.Run("walls_can_still_block", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Wall of Stone") // 0/8 Wall
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Invisibility")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Invisibility", "Grizzly Bears")
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		g.Block(1, mage.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Wall of Stone IS a Wall, so it can block invisible creature.
		// No damage to PlayerB.
		g.AssertLife(mage.PlayerB, 20)
	})
}

func TestLure(t *testing.T) {
	t.Run("all_creatures_must_block", func(t *testing.T) {
		// Lure: All creatures able to block enchanted creature do so.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Savannah Lions") // 2/1
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Gray Ogre")      // 2/2
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lure")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lure", "Grizzly Bears")
		g.Attack(1, mage.PlayerA, "Grizzly Bears", "Hill Giant")
		// Even though PlayerB tries to block Hill Giant, Lure forces blocks on Bears.
		g.Block(1, mage.PlayerB, "Savannah Lions", "Hill Giant")
		g.Block(1, mage.PlayerB, "Gray Ogre", "Hill Giant")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// All of PlayerB's creatures must block the Lured creature (Grizzly Bears),
		// not Hill Giant. So Hill Giant gets through unblocked for 3 damage.
		g.AssertLife(mage.PlayerB, 17)
	})
}

func TestWildGrowth(t *testing.T) {
	t.Run("enchanted_land_produces_extra_G", func(t *testing.T) {
		// Wild Growth: Whenever enchanted land is tapped for mana, its controller
		// adds an additional {G}.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Forest")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Wild Growth")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Wild Growth", "Forest")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Forest")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Auto-mana adds {G} to pay for Wild Growth. Forest taps for {G},
		// Wild Growth adds another {G}. Total: 1 (auto) + 1 (Forest) + 1 (bonus) = 3G.
		// Without bonus: 1 + 1 = 2G.
		pool := g.Players[0].ManaPool()
		if pool.Count(mage.Green) < 3 {
			t.Errorf("Wild Growth should add extra {G} when Forest taps; expected >= 3 green, got %d", pool.Count(mage.Green))
		}
	})
}

func TestCircleOfProtection(t *testing.T) {
	t.Run("prevents_red_damage", func(t *testing.T) {
		// Circle of Protection: Red -- {1}: The next time a red source of your
		// choice would deal damage to you this turn, prevent that damage.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Circle of Protection: Red")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lightning Bolt") // red source
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerB, "Circle of Protection: Red")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lightning Bolt", "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// CoP:Red should prevent the 3 red damage.
		g.AssertLife(mage.PlayerB, 20)
	})

	t.Run("doesnt_prevent_other_colors", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Circle of Protection: Red")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Terror")           // black spell
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // non-black creature for Terror
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hypnotic Specter") // black 2/2 flyer
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerB, "Circle of Protection: Red")
		g.Attack(1, mage.PlayerA, "Hypnotic Specter")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Hypnotic Specter is black, not red. CoP:Red shouldn't prevent its damage.
		g.AssertLife(mage.PlayerB, 18)
	})
}

func TestColorWards(t *testing.T) {
	t.Run("grants_protection_from_color", func(t *testing.T) {
		// Black Ward: Enchanted creature has protection from black.
		// (Testing one representative of the cycle.)
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Black Ward")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Terror") // black spell targeting non-black creature
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Black Ward", "Grizzly Bears")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Terror", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Protection from black: can't be targeted by Terror (black).
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("ward_not_removed_by_protection", func(t *testing.T) {
		// Special rule: Wards grant protection but don't fall off due to
		// protection from their own color (they have a special exception).
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Black Ward")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Black Ward", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Black Ward should still be attached even though it grants protection from black.
		g.AssertPermanentCount(mage.PlayerA, "Black Ward", 1)
		g.AssertAttachedTo(mage.PlayerA, "Black Ward", "Grizzly Bears")
	})
}

func TestLaceCycle(t *testing.T) {
	t.Run("changes_permanent_color", func(t *testing.T) {
		// Deathlace: Target spell or permanent becomes black.
		// (Testing one representative of the cycle.)
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // green
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Deathlace")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Deathlace", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Grizzly Bears should now be black.
		perm := g.FindPermanentByName("Grizzly Bears", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Grizzly Bears not found")
		}
		hasBlack := false
		for _, col := range perm.Colors() {
			if col == mage.Black {
				hasBlack = true
			}
		}
		if !hasBlack {
			t.Errorf("Deathlace should make Grizzly Bears black")
		}
	})
}
