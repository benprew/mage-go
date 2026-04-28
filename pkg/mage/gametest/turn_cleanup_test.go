// Package gametest: turn-structure tests for the cleanup step.
// Covers CR 514 (cleanup step): 514.1 (discard to maximum hand size), 514.2
// (remove marked damage / end-of-turn effects expire), 514.3 (no priority
// during cleanup), and 514.3a (extra cleanup when triggers resolve during
// cleanup).
package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestCleanup covers CR 514 sub-rules governing the cleanup step.
func TestCleanup(t *testing.T) {

	// CR 514.1 — If a player has more cards in their hand than their maximum
	// hand size (normally seven), that player discards enough cards to reduce
	// their hand size to that number. This happens as the first action of the
	// cleanup step.
	t.Run("CR 514.1 active player discards down to seven during cleanup", func(t *testing.T) {
		g := NewTestGame(t)
		for range 8 {
			g.AddCard(core.ZoneHand, PlayerA, "Forest")
		}
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertHandCount(PlayerA, "Forest", 7)
	})

	// CR 514.1 (negative) — Only the active player discards at cleanup. The
	// non-active player retains their full hand even if it is over seven cards.
	// (Overlaps existing coverage in TestCleanupDiscardOnlyActivePlayer; kept
	// here for completeness of the CR 514 suite.)
	t.Run("CR 514.1 non-active player does not discard during active player cleanup", func(t *testing.T) {
		g := NewTestGame(t)
		for range 9 {
			g.AddCard(core.ZoneHand, PlayerB, "Forest")
		}
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertHandCount(PlayerB, "Forest", 9)
	})

	// CR 514.2 — All damage marked on permanents is removed as part of the
	// cleanup step. A 3/3 that took 2 damage this turn must have zero marked
	// damage at the start of the next turn.
	t.Run("CR 514.2 marked damage on creatures is removed during cleanup", func(t *testing.T) {
		attacker := "ECU Marked Damage Attacker"
		blocker := "ECU Marked Damage Blocker"
		if !mage.CardRegistered(attacker) {
			mage.Register(attacker, func() mage.Card {
				return mage.NewCreature(attacker, "{2}{G}", 3, 3, mage.WithSubTypes("Beast"))
			})
		}
		if !mage.CardRegistered(blocker) {
			mage.Register(blocker, func() mage.Card {
				return mage.NewCreature(blocker, "{1}{W}", 1, 2, mage.WithSubTypes("Soldier"))
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, attacker)
		tg.AddCard(core.ZoneBattlefield, PlayerB, blocker)
		tg.Attack(1, PlayerA, attacker)
		tg.Block(1, PlayerB, blocker, attacker)
		// Stop during turn 2 upkeep — cleanup of turn 1 has run.
		tg.StopAt(2, core.Upkeep)
		tg.Execute()

		// Attacker (3/3) took 2 from the 1/2 blocker, survives, blocker died.
		tg.AssertPermanentCount(PlayerA, attacker, 1)
		tg.AssertPermanentCount(PlayerB, blocker, 0)

		// Inspect marked damage directly — must be zero after cleanup.
		perm := tg.FindPermanentByName(attacker, tg.GetPlayer(PlayerA).PlayerID())
		if perm == nil {
			t.Fatalf("attacker permanent not found")
		}
		if perm.Damage != 0 {
			t.Errorf("CR 514.2: expected marked damage 0 after cleanup, got %d", perm.Damage)
		}
	})

	// CR 514.2 — "Until end of turn" and "this turn" effects end as part of the
	// cleanup step. A 1/1 boosted to 3/3 "until end of turn" on turn 1 must be
	// a 1/1 again in turn 2's precombat main phase.
	t.Run("CR 514.2 until-end-of-turn effects expire during cleanup", func(t *testing.T) {
		creature := "ECU EOT Pump Target"
		pump := "ECU EOT Pump Spell"
		if !mage.CardRegistered(creature) {
			mage.Register(creature, func() mage.Card {
				return mage.NewCreature(creature, "{G}", 1, 1, mage.WithSubTypes("Elf"))
			})
		}
		if !mage.CardRegistered(pump) {
			mage.Register(pump, func() mage.Card {
				return mage.NewInstant(pump, "{G}",
					mage.NewTargetedSpell(
						mage.TargetCreature(),
						mage.BoostUntilEndOfTurn(mage.Fixed(2), mage.Fixed(2), mage.SelectTarget),
					),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, creature)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest", 1)
		tg.AddCard(core.ZoneHand, PlayerA, pump)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, pump, creature)

		// Stop at turn 2 precombat main — turn 1's cleanup has run, boost is gone.
		tg.StopAt(2, core.PrecombatMain)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, creature, 1, 1)
	})

	// CR 514.2 (sanity) — Confirm the "until end of turn" boost is observably
	// active before cleanup runs. This guards the above test from trivially
	// passing when the boost was never applied in the first place.
	t.Run("CR 514.2 until-end-of-turn boost is active before cleanup", func(t *testing.T) {
		creature := "ECU EOT Pump Target Sanity"
		pump := "ECU EOT Pump Spell Sanity"
		if !mage.CardRegistered(creature) {
			mage.Register(creature, func() mage.Card {
				return mage.NewCreature(creature, "{G}", 1, 1, mage.WithSubTypes("Elf"))
			})
		}
		if !mage.CardRegistered(pump) {
			mage.Register(pump, func() mage.Card {
				return mage.NewInstant(pump, "{G}",
					mage.NewTargetedSpell(
						mage.TargetCreature(),
						mage.BoostUntilEndOfTurn(mage.Fixed(2), mage.Fixed(2), mage.SelectTarget),
					),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, creature)
		tg.AddCard(core.ZoneHand, PlayerA, pump)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, pump, creature)
		tg.StopAt(1, core.EndStep)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, creature, 3, 3)
	})

	// CR 514.3 — Normally, no player receives priority during the cleanup step.
	// Observable: Game.CleanupPriorityRounds is incremented only when priority
	// is granted during a cleanup (CR 514.3a fallback). In a plain turn with
	// no cleanup-step triggers, it must stay at zero.
	t.Run("CR 514.3 no priority granted during cleanup in the normal case", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.StopAt(3, core.Upkeep)
		tg.Execute()
		if tg.CleanupPriorityRounds != 0 {
			t.Errorf("CR 514.3: expected CleanupPriorityRounds=0 with no cleanup triggers, got %d",
				tg.CleanupPriorityRounds)
		}
	})

	// CR 514.3a — If a triggered ability triggers during the cleanup step, the
	// active player receives priority after it is put on the stack, and after
	// the stack empties another cleanup step begins. Exercised here with a
	// permanent whose ability triggers "at the beginning of each cleanup
	// step," gated by an internal flag so it fires exactly once.
	t.Run("CR 514.3a trigger during cleanup grants priority and starts a new cleanup", func(t *testing.T) {
		cardName := "ECU Cleanup Trigger Source"
		fired := false
		if !mage.CardRegistered(cardName) {
			mage.Register(cardName, func() mage.Card {
				return mage.NewEnchantment(cardName, "{1}",
					mage.WithAbility(
						mage.BeginningOfEachCleanupStepTrigger(
							mage.FuncEffect("gain 1 life once during cleanup",
								mage.EffectProperties{Outcome: mage.OutcomeBenefit},
								func(g *mage.Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
									p := g.GetPlayer(controller)
									if p != nil {
										g.PlayerGainLife(p, 1)
									}
									return nil
								}), false,
						).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, controllerID uuid.UUID) bool {
							if fired || evt.PlayerID != controllerID {
								return false
							}
							fired = true
							return true
						}),
					),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
		tg.StopAt(2, core.Upkeep)
		tg.Execute()

		tg.AssertLife(PlayerA, 21)
		if tg.CleanupPriorityRounds != 1 {
			t.Errorf("CR 514.3a: expected exactly one cleanup priority round, got %d",
				tg.CleanupPriorityRounds)
		}
	})
}

// TestCleanupDiscardOnlyActivePlayer (moved from mechanics_test.go) verifies
// CR 514.1 — only the active player performs the discard-to-maximum-hand-size
// action during cleanup. The "non-active player does not discard" sub-test is
// a close duplicate of the same-named sub-test in TestCleanup above but uses
// a hand size of 8 rather than 9 — we keep both because they exercise subtly
// different boundary conditions.
func TestCleanupDiscardOnlyActivePlayer(t *testing.T) {
	t.Run("non-active player does not discard during active player cleanup", func(t *testing.T) {
		g := NewTestGame(t)
		for range 8 {
			g.AddCard(core.ZoneHand, PlayerB, "Forest")
		}
		// Stop after turn 1 cleanup has executed
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertHandCount(PlayerB, "Forest", 8)
	})

	t.Run("active player discards during their own cleanup", func(t *testing.T) {
		g := NewTestGame(t)
		for range 8 {
			g.AddCard(core.ZoneHand, PlayerA, "Forest")
		}
		// Stop after turn 1 cleanup has executed
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertHandCount(PlayerA, "Forest", 7)
	})
}
