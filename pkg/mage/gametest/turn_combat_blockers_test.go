// Package gametest: turn-structure tests for the declare-blockers step.
// Covers CR 509.1a (tapped creatures cannot block) and CR 509.1h (blocked
// creature stays blocked when blockers are removed from combat).
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestDeclareBlockers covers CR 509 sub-rules governing the declare-blockers
// step.
func TestDeclareBlockers(t *testing.T) {

	// CR 509.1a — The defending player chooses which of their untapped
	// creatures will block. A tapped creature can't be declared as a blocker.
	t.Run("CR 509.1a tapped creature cannot block", func(t *testing.T) {
		attacker := "Combat Block-Test Attacker"
		blocker := "Combat Block-Test Blocker"
		if !mage.CardRegistered(attacker) {
			mage.Register(attacker, func() mage.Card {
				return mage.NewCreature(attacker, "{1}{R}", 2, 2, mage.WithSubTypes("Ogre"))
			})
		}
		if !mage.CardRegistered(blocker) {
			mage.Register(blocker, func() mage.Card {
				return mage.NewCreature(blocker, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, attacker)
		permID := tg.AddCard(core.ZoneBattlefield, PlayerB, blocker)
		perm := tg.FindPermanent(permID)
		if perm == nil {
			t.Fatalf("blocker permanent not found")
		}
		perm.Tapped = true
		tg.Attack(1, PlayerA, attacker)
		tg.Block(1, PlayerB, blocker, attacker)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Tapped blocker can't block → full damage through.
		tg.AssertLife(PlayerB, 18)
		tg.AssertPermanentCount(PlayerB, blocker, 1) // blocker survives, didn't deal or take damage
	})

	// CR 509.1h — A blocked creature remains blocked even if all creatures
	// blocking it are removed from combat. Observable: the attacker's combat
	// damage to the defending player is 0 (it deals damage only to blockers
	// or, if unblocked, to the defender; blocked-with-no-blockers still means
	// blocked).
	//
	// This scenario is covered by TestTurnStructureCombatRemoval's CR 509.1h
	// test (turn_combat_test.go). Skip here to avoid duplication.
	t.Run("CR 509.1h blocked-then-blocker-destroyed attacker deals 0 to player", func(t *testing.T) {
		t.Skip("covered by Batch C (turn_structure_combat_removal_test.go: CR 509.1h)")
	})
}
