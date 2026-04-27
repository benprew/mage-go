package limited

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for creature cards registered in alpha_creatures.go.

func TestThicketBasilisk(t *testing.T) {
	t.Run("destroys_blocking_non_wall_creature", func(t *testing.T) {
		// Thicket Basilisk: Whenever Thicket Basilisk blocks or becomes blocked
		// by a non-Wall creature, destroy that creature at end of combat.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thicket Basilisk") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")        // 6/4
		g.Attack(1, gametest.PlayerA, "Thicket Basilisk")
		g.Block(1, gametest.PlayerB, "Craw Wurm", "Thicket Basilisk")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Basilisk deals 2 damage to Craw Wurm. Basilisk ability destroys
		// the non-Wall creature at end of combat.
		g.AssertGraveyardCount(gametest.PlayerB, "Craw Wurm", 1)
	})

	t.Run("doesnt_destroy_walls", func(t *testing.T) {
		// Basilisk ability specifically excludes Walls.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Thicket Basilisk") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")       // 3/3 attacker
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone")    // 0/8 Wall
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Hill Giant")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Wall of Stone blocks Hill Giant normally. Basilisk is not involved.
		// Now test Basilisk blocking a Wall:
		g2 := gametest.NewTestGame(t)
		g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Juggernaut")       // 5/3 can't be blocked by Walls, but let's use a different attacker
		g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")        // 6/4
		g2.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Thicket Basilisk") // 2/4
		g2.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone")    // 0/8 Wall
		g2.Attack(1, gametest.PlayerA, "Craw Wurm")
		g2.Block(1, gametest.PlayerB, "Thicket Basilisk", "Craw Wurm")
		g2.StopAt(1, core.PostcombatMain)
		g2.Execute()
		// Basilisk blocks Craw Wurm (non-Wall) -> Craw Wurm destroyed at end of combat.
		g2.AssertGraveyardCount(gametest.PlayerA, "Craw Wurm", 1)
		// Basilisk takes 6 damage but has 4 toughness -> dies to combat damage.
		// That's fine. The key test for "doesn't destroy walls" is below:

		g3 := gametest.NewTestGame(t)
		g3.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Stone")    // 0/8 Wall with Defender
		g3.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Thicket Basilisk") // 2/4
		// Basilisk attacks, Wall blocks it
		g3.Attack(2, gametest.PlayerB, "Thicket Basilisk")
		g3.Block(2, gametest.PlayerA, "Wall of Stone", "Thicket Basilisk")
		g3.StopAt(2, core.PostcombatMain)
		g3.Execute()
		// Wall of Stone IS a Wall. Basilisk ability should NOT destroy it.
		g3.AssertPermanentCount(gametest.PlayerA, "Wall of Stone", 1)
	})

	t.Run("destroys_creature_it_blocks", func(t *testing.T) {
		// Basilisk destroys non-Wall creatures that it blocks, too.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")        // 6/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Thicket Basilisk") // 2/4
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.Block(1, gametest.PlayerB, "Thicket Basilisk", "Craw Wurm")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Basilisk blocks Craw Wurm. Craw Wurm deals 6 to Basilisk (kills it).
		// Basilisk ability destroys Craw Wurm (non-Wall) at end of combat.
		g.AssertGraveyardCount(gametest.PlayerA, "Craw Wurm", 1)
	})
}

func TestBirdsOfParadise(t *testing.T) {
	t.Run("taps_for_any_color", func(t *testing.T) {
		// Birds of Paradise: {T}: Add one mana of any color.
		colors := []struct {
			name  string
			color core.Color
		}{
			{"white", core.White},
			{"blue", core.Blue},
			{"black", core.Black},
			{"red", core.Red},
			{"green", core.Green},
		}

		for _, tc := range colors {
			t.Run(tc.name, func(t *testing.T) {
				g := gametest.NewTestGame(t)
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Birds of Paradise")
				g.ChooseManaColor(gametest.PlayerA, tc.color)
				g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Birds of Paradise")
				g.StopAt(1, core.BeginCombat)
				g.Execute()
				// Auto-mana gives 5 of each color. Birds adds 1 of chosen color.
				// For non-green colors, count should be 6 if Birds works correctly.
				pool := g.AllPlayers()[0].ManaPool()
				if tc.color != core.Green {
					if pool.CountProducedThisTurn(tc.color) < 6 {
						t.Errorf("Birds of Paradise should produce %s mana; pool has %d, want >= 6",
							tc.name, pool.CountProducedThisTurn(tc.color))
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
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel") // 4/4 flying, vigilance
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Vesuvan Doppelganger")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Vesuvan Doppelganger", "Serra Angel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Doppelganger should enter as a copy of Serra Angel: 4/4 with flying.
		g.AssertPermanentCount(gametest.PlayerA, "Vesuvan Doppelganger", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Vesuvan Doppelganger", 4, 4)
		g.AssertHasAbility(gametest.PlayerA, "Vesuvan Doppelganger", core.Flying, true)
	})

	t.Run("can_change_copy_at_upkeep", func(t *testing.T) {
		// At the beginning of your upkeep, you may have Doppelganger become a
		// copy of a different creature.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")   // 4/4 flying
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shivan Dragon") // 5/5 flying
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Vesuvan Doppelganger")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Vesuvan Doppelganger", "Serra Angel")
		// Turn 3 is PlayerA's next turn. At upkeep, Doppelganger can change to Shivan Dragon.
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// After upkeep trigger, Doppelganger should now be a copy of Shivan Dragon: 5/5.
		g.AssertPermanentCount(gametest.PlayerA, "Vesuvan Doppelganger", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Vesuvan Doppelganger", 5, 5)
	})
}

func TestScavengingGhoul(t *testing.T) {
	t.Run("gains_corpse_counters_when_creatures_die", func(t *testing.T) {
		// Two creatures die in combat, Scavenging Ghoul should get 2 corpse counters at end step.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scavenging Ghoul")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		// Both Hill Giants trade in combat
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Hill Giant")
		g.StopAt(2, core.Upkeep) // stop after end step triggers resolve
		g.Execute()
		// Both Hill Giants died, so Scavenging Ghoul gets 2 corpse counters
		g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
		g.AssertCounterCount(gametest.PlayerA, "Scavenging Ghoul", core.Corpse, 2)
	})

	t.Run("no_counters_when_no_creatures_die", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scavenging Ghoul")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Scavenging Ghoul", core.Corpse, 0)
	})

	t.Run("regenerate_with_corpse_counter", func(t *testing.T) {
		// Give Scavenging Ghoul a corpse counter, then use it to regenerate when it would die.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scavenging Ghoul") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")       // 3/3
		g.AddCounters(1, core.Upkeep, gametest.PlayerA, "Scavenging Ghoul", core.Corpse, 1)
		// Activate regeneration before combat
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Scavenging Ghoul")
		// Attack with Hill Giant into Scavenging Ghoul as blocker
		g.Attack(1, gametest.PlayerB, "Hill Giant")
		g.Block(1, gametest.PlayerA, "Scavenging Ghoul", "Hill Giant")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Scavenging Ghoul should survive via regeneration (corpse counter removed)
		g.AssertPermanentCount(gametest.PlayerA, "Scavenging Ghoul", 1)
		g.AssertCounterCount(gametest.PlayerA, "Scavenging Ghoul", core.Corpse, 0)
	})
}
