// Package gametest: turn-structure tests for the combat phase (general rules
// and removal). Covers CR 506 (combat phase general, removal from combat),
// CR 507 (beginning-of-combat step — via EvtBeginCombat in 506.1), and CR 511
// (combat-phase terminators: 511.2 end-of-combat triggers, 511.3 combat
// cleanup). Declare-attackers / declare-blockers / damage-step details live
// in turn_combat_attackers_test.go, turn_combat_blockers_test.go, and
// turn_combat_damage_test.go.
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// registerRemovalBolt registers an instant that deals 3 damage to a target
// creature, used by several of the removal-from-combat tests below.
func registerRemovalBolt(name string) {
	if mage.CardRegistered(name) {
		return
	}
	mage.Register(name, func() mage.Card {
		return mage.NewInstant(name, "{R}",
			mage.NewTargetedSpell(mage.TargetCreature(), mage.DealDamage(mage.Fixed(3))))
	})
}

// registerRemovalTapRay registers an instant that taps a target creature.
// Used to verify CR 506.4b — tapping an already-declared attacker does not
// remove it from combat.
func registerRemovalTapRay(name string) {
	if mage.CardRegistered(name) {
		return
	}
	mage.Register(name, func() mage.Card {
		return mage.NewInstant(name, "{U}",
			mage.NewTargetedSpell(mage.TargetCreature(), mage.TapTarget()))
	})
}

// registerRemovalBear registers a vanilla creature with configurable P/T under
// the given name. No summoning-sickness concerns because AddCard revokes
// AttrSummonSick when placing cards on the battlefield.
func registerRemovalBear(name string, power, toughness int) {
	if mage.CardRegistered(name) {
		return
	}
	mage.Register(name, func() mage.Card {
		return mage.NewCreature(name, "{2}{G}", power, toughness, mage.WithSubTypes("Bear"))
	})
}

// TestCombatPhaseGeneral covers the combat phase's top-level structure and
// terminators: CR 506.1 (step order), CR 511.2 (at-end-of-combat triggers),
// and CR 511.3 (creatures removed from combat after EndCombat).
func TestCombatPhaseGeneral(t *testing.T) {

	// CR 506.1 — The combat phase has five steps, which occur in this order:
	// beginning of combat, declare attackers, declare blockers, combat damage,
	// and end of combat. The declare blockers and combat damage steps are
	// skipped if no creatures are declared as attackers or put onto the
	// battlefield attacking.
	//
	// We attach triggers listening for each of the combat-phase events to a
	// single permanent, then attack with it and assert that the events fire in
	// the canonical order.
	t.Run("CR 506.1 combat phase steps fire in order", func(t *testing.T) {
		registerCombatEventRecorder()
		combatRec.reset()

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Combat Event Recorder")
		tg.Attack(1, PlayerA, "Combat Event Recorder")
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		got := combatRec.snapshot()
		want := []core.EventType{
			core.EvtBeginCombat,
			core.EvtDeclaredAttacker,
			core.EvtBlockersDecl,
			core.EvtEndOfCombat,
		}
		if len(got) != len(want) {
			t.Fatalf("event count: got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("event %d: got %s, want %s", i, got[i], want[i])
			}
		}
	})

	// CR 511.2 — Abilities that trigger "at end of combat" trigger as the
	// end-of-combat step begins. We attach a simple EvtEndOfCombat trigger
	// that gains life; if it fires exactly once, the life total changed.
	t.Run("CR 511.2 at-end-of-combat triggers fire during end-of-combat step", func(t *testing.T) {
		name := "Combat EOC Lifegain"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{2}", 1, 1,
					mage.WithSubTypes("Spirit"),
					mage.WithAbility(mage.NewTriggered(core.EvtEndOfCombat, false,
						mage.GainLife(3))))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerA, 23)
	})

	// CR 511.3 — "As soon as the end of combat step ends, all creatures and
	// planeswalkers are removed from combat." Engine-wise, priority.go calls
	// g.Combat.Reset() at the end of the EndCombat step. We drive a normal
	// combat, stop at PostcombatMain (so EndCombat has already executed), and
	// assert that no permanent is reported as attacking.
	t.Run("CR 511.3 creatures removed from combat after end-of-combat step", func(t *testing.T) {
		attacker := "ECU2 511.3 Attacker"
		if !mage.CardRegistered(attacker) {
			mage.Register(attacker, func() mage.Card {
				return mage.NewCreature(attacker, "{2}{G}", 3, 3, mage.WithSubTypes("Beast"))
			})
		}

		tg := NewTestGame(t)
		attackerID := tg.AddCard(core.ZoneBattlefield, PlayerA, attacker)
		tg.Attack(1, PlayerA, attacker)
		// Stop at the PostcombatMain of turn 1 — EndCombat (which resets
		// combat) has already run.
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		if tg.Combat.IsAttacking(attackerID) {
			t.Errorf("CR 511.3: attacker still reported as attacking after EndCombat step; want removed from combat")
		}
		// Sanity: walk every permanent and confirm none is marked attacking.
		for _, perm := range tg.Battlefield {
			if tg.Combat.IsAttacking(perm.ID()) {
				t.Errorf("CR 511.3: %s still attacking after EndCombat", perm.Name())
			}
		}
	})
}

// TestTurnStructureCombatRemoval covers CR 506.4 and its subrules governing
// how creatures are (or are not) removed from combat after the declare-
// attackers / declare-blockers steps.
func TestTurnStructureCombatRemoval(t *testing.T) {

	// CR 506.4 — A creature is removed from combat when it leaves the
	// battlefield. An attacker destroyed during the declare-blockers step is
	// no longer on the battlefield when the combat damage step begins, so it
	// deals no combat damage.
	t.Run("CR 506.4 destroyed attacker deals no damage", func(t *testing.T) {
		attacker := "Removal 506.4 Attacker 3/3"
		bolt := "Removal 506.4 Bolt"
		registerRemovalBear(attacker, 3, 3)
		registerRemovalBolt(bolt)

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, attacker)
		tg.AddCard(core.ZoneHand, PlayerB, bolt)
		tg.Attack(1, PlayerA, attacker)
		// Bolt the attacker mid-combat (during declare-blockers) so it leaves
		// the battlefield before the combat damage step.
		tg.CastSpell(1, core.DeclareBlockers, PlayerB, bolt, attacker)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		tg.AssertLife(PlayerB, 20) // attacker gone before damage step
		tg.AssertPermanentCount(PlayerA, attacker, 0)
		tg.AssertGraveyardCount(PlayerA, attacker, 1)
	})

	// CR 506.4 / CR 509.1h — A blocked creature remains blocked even if all
	// creatures blocking it are removed from combat (by being destroyed, in
	// this case). Without trample, the attacker deals no damage to the
	// defending player.
	t.Run("CR 509.1h attacker still blocked after blocker destroyed", func(t *testing.T) {
		attacker := "Removal 509.1h Attacker 3/3"
		blocker := "Removal 509.1h Blocker 2/2"
		bolt := "Removal 509.1h Bolt"
		registerRemovalBear(attacker, 3, 3)
		registerRemovalBear(blocker, 2, 2)
		registerRemovalBolt(bolt)

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, attacker)
		tg.AddCard(core.ZoneBattlefield, PlayerB, blocker)
		tg.AddCard(core.ZoneHand, PlayerA, bolt)
		tg.Attack(1, PlayerA, attacker)
		tg.Block(1, PlayerB, blocker, attacker)
		// Per CR 509.1h the attacker remains blocked even if its sole blocker
		// leaves combat; without trample it deals 0 damage to the defending
		// player. Scheduled at CombatDamage (after blocks are locked in but
		// before damage resolves); see turn_structure_status.md on the harness
		// ordering quirk that prevents scheduling at DeclareBlockers here.
		tg.CastSpell(1, core.CombatDamage, PlayerA, bolt, blocker)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		tg.AssertLife(PlayerB, 20)
		tg.AssertGraveyardCount(PlayerB, blocker, 1)
		tg.AssertPermanentCount(PlayerA, attacker, 1)
	})

	// CR 506.4a — An effect that would prevent a creature from attacking,
	// applied after it has already been declared as an attacker, does not
	// remove it from combat.
	//
	// The engine exposes only an attachment-based PreventAttachedFromAttacking
	// continuous effect; there is no instant-speed targeted "creature can't
	// attack this turn" card. This test is skipped until such an effect is
	// available at instant speed.
	t.Run("CR 506.4a post-declaration cant-attack does not remove attacker", func(t *testing.T) {
		t.Skip("XXX: engine gap — no runtime instant-speed 'cant attack' effect to apply mid-combat")
	})

	// CR 506.4b — Tapping a creature that's already been declared as an
	// attacker does not remove it from combat, and that creature still deals
	// its combat damage.
	t.Run("CR 506.4b tapping already-declared attacker does not remove it from combat", func(t *testing.T) {
		attacker := "Removal 506.4b Attacker 3/3"
		tapRay := "Removal 506.4b Tap Ray"
		registerRemovalBear(attacker, 3, 3)
		registerRemovalTapRay(tapRay)

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, attacker)
		tg.AddCard(core.ZoneHand, PlayerB, tapRay)
		tg.Attack(1, PlayerA, attacker)
		// Tap the already-declared attacker during the declare-blockers step.
		// Attacking already taps it, so this is idempotent for the "tapped"
		// state, but the point is that the "tap" action mid-combat does not
		// remove the attacker from combat.
		tg.CastSpell(1, core.DeclareBlockers, PlayerB, tapRay, attacker)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		tg.AssertTapped(PlayerA, attacker, true)
		tg.AssertLife(PlayerB, 17) // attacker still deals its 3 damage
		tg.AssertPermanentCount(PlayerA, attacker, 1)
	})

	// CR 506.5 — "Attacks alone" / "attacking alone" / "blocks alone" /
	// "blocking alone". The engine does not currently expose a selector or
	// trigger that distinguishes attacking-alone from attacking-with-others.
	// Skipped until such a selector exists.
	t.Run("CR 506.5 attacks-alone / blocks-alone selector", func(t *testing.T) {
		t.Skip("XXX: engine gap — no 'attacks alone' / 'blocks alone' selector or trigger condition")
	})
}
