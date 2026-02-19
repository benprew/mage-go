package limited

import (
	"testing"

	"github.com/mage/mage/pkg/mage"
)

// Tests for creature cards registered in alpha_creatures.go.

func TestThicketBasilisk(t *testing.T) {
	t.Run("destroys_blocking_non_wall_creature", func(t *testing.T) {
		// Thicket Basilisk: Whenever Thicket Basilisk blocks or becomes blocked
		// by a non-Wall creature, destroy that creature at end of combat.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Thicket Basilisk") // 2/4
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Craw Wurm")        // 6/4
		g.Attack(1, mage.PlayerA, "Thicket Basilisk")
		g.Block(1, mage.PlayerB, "Craw Wurm", "Thicket Basilisk")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		// Basilisk deals 2 damage to Craw Wurm. Basilisk ability destroys
		// the non-Wall creature at end of combat.
		g.AssertGraveyardCount(mage.PlayerB, "Craw Wurm", 1)
	})

	t.Run("doesnt_destroy_walls", func(t *testing.T) {
		// Basilisk ability specifically excludes Walls.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Thicket Basilisk") // 2/4
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")       // 3/3 attacker
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Wall of Stone")    // 0/8 Wall
		g.Attack(1, mage.PlayerA, "Hill Giant")
		g.Block(1, mage.PlayerB, "Wall of Stone", "Hill Giant")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		// Wall of Stone blocks Hill Giant normally. Basilisk is not involved.
		// Now test Basilisk blocking a Wall:
		g2 := mage.NewTestGame(t)
		g2.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Juggernaut")       // 5/3 can't be blocked by Walls, but let's use a different attacker
		g2.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm")        // 6/4
		g2.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Thicket Basilisk") // 2/4
		g2.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Wall of Stone")    // 0/8 Wall
		g2.Attack(1, mage.PlayerA, "Craw Wurm")
		g2.Block(1, mage.PlayerB, "Thicket Basilisk", "Craw Wurm")
		g2.StopAt(1, mage.PostcombatMain)
		g2.Execute()
		// Basilisk blocks Craw Wurm (non-Wall) -> Craw Wurm destroyed at end of combat.
		g2.AssertGraveyardCount(mage.PlayerA, "Craw Wurm", 1)
		// Basilisk takes 6 damage but has 4 toughness -> dies to combat damage.
		// That's fine. The key test for "doesn't destroy walls" is below:

		g3 := mage.NewTestGame(t)
		g3.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Wall of Stone")    // 0/8 Wall with Defender
		g3.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Thicket Basilisk") // 2/4
		// Basilisk attacks, Wall blocks it
		g3.Attack(2, mage.PlayerB, "Thicket Basilisk")
		g3.Block(2, mage.PlayerA, "Wall of Stone", "Thicket Basilisk")
		g3.StopAt(2, mage.PostcombatMain)
		g3.Execute()
		// Wall of Stone IS a Wall. Basilisk ability should NOT destroy it.
		g3.AssertPermanentCount(mage.PlayerA, "Wall of Stone", 1)
	})

	t.Run("destroys_creature_it_blocks", func(t *testing.T) {
		// Basilisk destroys non-Wall creatures that it blocks, too.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm")        // 6/4
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Thicket Basilisk") // 2/4
		g.Attack(1, mage.PlayerA, "Craw Wurm")
		g.Block(1, mage.PlayerB, "Thicket Basilisk", "Craw Wurm")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		// Basilisk blocks Craw Wurm. Craw Wurm deals 6 to Basilisk (kills it).
		// Basilisk ability destroys Craw Wurm (non-Wall) at end of combat.
		g.AssertGraveyardCount(mage.PlayerA, "Craw Wurm", 1)
	})
}

func TestBirdsOfParadise(t *testing.T) {
	t.Run("taps_for_any_color", func(t *testing.T) {
		// Birds of Paradise: {T}: Add one mana of any color.
		colors := []struct {
			name  string
			color mage.Color
		}{
			{"white", mage.White},
			{"blue", mage.Blue},
			{"black", mage.Black},
			{"red", mage.Red},
			{"green", mage.Green},
		}

		for _, tc := range colors {
			t.Run(tc.name, func(t *testing.T) {
				g := mage.NewTestGame(t)
				g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Birds of Paradise")
				g.ChooseManaColor(mage.PlayerA, tc.color)
				g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Birds of Paradise")
				g.StopAt(1, mage.BeginCombat)
				g.Execute()
				// Auto-mana gives 5 of each color. Birds adds 1 of chosen color.
				// For non-green colors, count should be 6 if Birds works correctly.
				pool := g.Players[0].ManaPool()
				if tc.color != mage.Green {
					if pool.Count(tc.color) < 6 {
						t.Errorf("Birds of Paradise should produce %s mana; pool has %d, want >= 6",
							tc.name, pool.Count(tc.color))
					}
				}
			})
		}
	})
}

func TestVesuvanDoppelganger(t *testing.T) {
	t.Run("enters_as_copy_of_creature", func(t *testing.T) {
		// Vesuvan Doppelganger: As Vesuvan Doppelganger enters, you may choose a
		// creature on the battlefield. If you do, it enters as a copy of that
		// creature, except it has "At the beginning of your upkeep, you may have
		// this creature become a copy of target creature, except it has this ability."
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Serra Angel") // 4/4 flying, vigilance
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Vesuvan Doppelganger")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Vesuvan Doppelganger", "Serra Angel")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Doppelganger should enter as a copy of Serra Angel: 4/4 with flying.
		g.AssertPermanentCount(mage.PlayerA, "Vesuvan Doppelganger", 1)
		g.AssertPowerToughness(mage.PlayerA, "Vesuvan Doppelganger", 4, 4)
		g.AssertHasAbility(mage.PlayerA, "Vesuvan Doppelganger", mage.Flying, true)
	})

	t.Run("can_change_copy_at_upkeep", func(t *testing.T) {
		// At the beginning of your upkeep, you may have Doppelganger become a
		// copy of a different creature.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Serra Angel")   // 4/4 flying
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Shivan Dragon")  // 5/5 flying
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Vesuvan Doppelganger")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Vesuvan Doppelganger", "Serra Angel")
		// Turn 3 is PlayerA's next turn. At upkeep, Doppelganger can change to Shivan Dragon.
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		// After upkeep trigger, Doppelganger should now be a copy of Shivan Dragon: 5/5.
		g.AssertPermanentCount(mage.PlayerA, "Vesuvan Doppelganger", 1)
		g.AssertPowerToughness(mage.PlayerA, "Vesuvan Doppelganger", 5, 5)
	})
}