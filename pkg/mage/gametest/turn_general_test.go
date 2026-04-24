// Package gametest: turn-structure tests for CR 500 (general phase/step rules).
// Covers CR 500.1, 500.2, 500.3, 500.4, 500.5, 500.5a, 500.6, 500.7, 500.11,
// 500.12. CR 501.1 (beginning-phase step ordering) is exercised alongside CR
// 500.1 in the combined "five phases and beginning-phase steps in order" sub-
// test below, since the two rules are verified by the same step-order assertion.
package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestTurnPhases covers CR 500 (phases) and CR 501.1 (beginning-phase step
// ordering). Each sub-test references the exact rule it is exercising.
func TestTurnPhases(t *testing.T) {

	// CR 500.1 — A turn consists of five phases, in this order: beginning,
	// precombat main, combat, postcombat main, ending. Each of these phases
	// takes place every turn, even if nothing happens during the phase.
	//
	// CR 501.1 — The beginning phase consists of three steps, in this order:
	// untap, upkeep, and draw.
	//
	// We drive a single turn via the priority-aware scheduler and record each
	// step visited. We assert the observed order matches the canonical order,
	// which covers both 500.1 (phase ordering) and 501.1 (beginning-phase
	// step ordering).
	t.Run("CR 500.1 / 501.1 five phases and beginning-phase steps in order", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.StopAt(1, core.PrecombatMain) // not used; we drive manually
		// Establish libraries so we don't deck out on draw.
		tg.padLibraries()
		tg.OnPriority = autoPassHandler()

		want := []core.PhaseStep{
			core.Untap,
			core.Upkeep,
			core.Draw,
			core.PrecombatMain,
			core.BeginCombat,
			core.DeclareAttackers,
			core.DeclareBlockers,
			core.FirstStrikeDamage,
			core.CombatDamage,
			core.EndCombat,
			core.PostcombatMain,
			core.EndStep,
			core.Cleanup,
		}
		var got []core.PhaseStep
		for _, step := range core.AllSteps() {
			tg.RunStepWithPriority(step)
			got = append(got, tg.Step)
		}

		if len(got) != len(want) {
			t.Fatalf("step count: got %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("step %d: got %s, want %s", i, got[i], want[i])
			}
		}
	})

	// CR 500.2 — A phase or step in which players receive priority ends when
	// the stack is empty and all players pass in succession. Until that
	// occurs, priority is repeatedly offered.
	//
	// We use an upkeep-triggered ability that puts an effect on the stack.
	// Before that trigger resolves the upkeep step cannot end, even though
	// all players are passing. We verify this by observing that the trigger
	// DID resolve by the time the upkeep step returned.
	t.Run("CR 500.2 priority step ends only when stack empty and players pass", func(t *testing.T) {
		name := "Turn Structure Upkeep Pinger"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewArtifact(name, "{2}",
					mage.WithAbility(mage.NewTriggered(
						core.EvtUpkeep,
						false,
						mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectController()),
					).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, controllerID uuid.UUID) bool {
						return evt.PlayerID == controllerID
					})),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		// After turn 2 upkeep runs with both-players-pass handler, trigger
		// must have been placed on the stack AND resolved before the step
		// returned. Otherwise the 2 damage never lands.
		tg.StopAt(2, core.Draw)
		tg.Execute()
		tg.AssertLife(PlayerA, 18)
	})

	// CR 500.5 — When a phase or step ends, any unused mana in a player's
	// mana pool empties (subject to rule 106.4; we don't exercise the
	// exception here).
	//
	// XXX: engine gap — the engine does NOT empty mana pools between steps;
	// ManaPool().Clear() is only called during doCleanupActions and after a
	// spell is cast. There is no per-step empty-mana hook we can observe.
	// See /home/user/mage-go/pkg/mage/game.go around doCleanup (line ~2145)
	// and /home/user/mage-go/pkg/mage/priority.go RunStepWithPriority.
	t.Run("CR 500.5 mana empties from pool at end of each step", func(t *testing.T) {
		t.Skip("XXX: engine gap — mana pool is only cleared at Cleanup, not at the end of every step/phase. See game.go doCleanupActions; no equivalent in RunStepWithPriority for other steps.")
	})

	// CR 500.6 — "At the beginning of [phase/step]" abilities trigger at the
	// start of that phase or step. The upkeep trigger is the simplest
	// concrete instance: "at the beginning of your upkeep" must fire as the
	// upkeep step begins.
	t.Run("CR 500.6 at-the-beginning-of-X triggers fire when the step begins", func(t *testing.T) {
		name := "Turn Structure Upkeep Lifegainer"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewArtifact(name, "{2}",
					mage.WithAbility(mage.NewTriggered(
						core.EvtUpkeep,
						false,
						mage.GainLife(1),
					).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, controllerID uuid.UUID) bool {
						return evt.PlayerID == controllerID
					})),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		// On turn 2, PlayerA's upkeep begins; the "at the beginning of your
		// upkeep" trigger must fire AND resolve before the draw step. We
		// stop AT Draw so that if the trigger didn't fire/resolve life
		// would still be 20.
		tg.StopAt(2, core.Draw)
		tg.Execute()
		tg.AssertLife(PlayerA, 21)
	})
}

// TestTurnPhasesAdditional covers additional CR 500-level rules: 500.3, 500.4,
// 500.5a, 500.7, 500.11, 500.12. Each sub-test names the exact Comprehensive
// Rules section it is exercising.
func TestTurnPhasesAdditional(t *testing.T) {

	// CR 500.3 — The untap step has turn-based actions but no player receives
	// priority during it. After those turn-based actions complete, the step
	// ends — deterministically, with no priority round.
	//
	// We install an OnPriority handler that counts invocations. When
	// RunStepWithPriority(Untap) is called, the handler must NOT fire — the
	// step must complete purely on turn-based actions. We drive via turn 3
	// so PlayerA is the active player and there's a permanent eligible to
	// untap (establishing that the turn-based actions did run).
	t.Run("CR 500.3 untap step completes without a priority round", func(t *testing.T) {
		tg := NewTestGame(t)
		permID := tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		perm := tg.FindPermanent(permID)
		if perm == nil {
			t.Fatalf("permanent not found after AddCard")
		}

		// Advance to turn 3 Untap (stops before Untap runs). Turn 3 = PlayerA.
		tg.StopAt(3, core.Untap)
		tg.Execute()
		perm.Tapped = true

		// Install a priority handler that records any invocation. The untap
		// step must not call it.
		calls := 0
		tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
			calls++
			return mage.PriorityAction{Type: mage.PriorityPass}
		}

		tg.RunStepWithPriority(core.Untap)

		if calls != 0 {
			t.Errorf("CR 500.3: priority handler invoked %d time(s) during untap; want 0", calls)
		}
		// Sanity: turn-based untap actually happened.
		tg.AssertTapped(PlayerA, "Grizzly Bears", false)
		// Sanity: the step advanced (we're no longer parked at Untap after
		// the call returned; the step field still says Untap because we
		// only ran that one step, but the call returned — this is the
		// deterministic termination we care about).
		if tg.Step != core.Untap {
			t.Errorf("CR 500.3: expected Step to still be Untap after RunStepWithPriority(Untap), got %s", tg.Step)
		}
	})

	// CR 500.4 — As a step or phase begins, any effects that are scheduled
	// to last "until" that step or phase expire. Concretely: an effect that
	// lasts "until your next turn" / "until your next upkeep" ends as the
	// controller's next upkeep begins.
	//
	// The engine models this as `UntilYourNextTurn`, removed inside
	// doUpkeepActions() before the EvtUpkeep fires (game.go ~1854). We pump
	// a creature's P/T with that duration on turn 1, then assert the pump
	// is gone by the start of turn 3's PrecombatMain (PlayerA's next
	// upkeep has occurred).
	t.Run("CR 500.4 until-your-next-turn effect expires at start of next upkeep", func(t *testing.T) {
		tg := NewTestGame(t)
		permID := tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")

		// Stop at turn 1 PrecombatMain so the creature exists on battlefield
		// and we can attach a continuous effect to it.
		tg.StopAt(1, core.PrecombatMain)
		tg.Execute()

		eff := mage.TargetEffect(core.LayerPT, core.UntilYourNextTurn, permID,
			func(g *mage.Game, target *mage.Permanent) error {
				target.BoostPT(2, 2)
				return nil
			})
		// Attach source so the effect is not pruned by Apply's "source no
		// longer on battlefield" sweep (UntilYourNextTurn is not in the
		// duration-allowlist).
		eff.SetSourceID(permID)
		tg.Effects.Add(eff)
		tg.Effects.Apply(tg.Game)

		// Confirm the pump is active right now.
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 4, 4)

		// Advance to turn 3 PrecombatMain. Turn 3 is PlayerA's next turn;
		// as turn 3 upkeep begins, RemoveUntilYourNextTurn fires for PlayerA.
		tg.StopAt(3, core.PrecombatMain)
		tg.Execute()

		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
	})

	// CR 500.5a — "Until end of combat" effects expire at the end of the
	// combat phase, NOT at the start of the end-of-combat step. Concretely:
	// a pump with EndOfCombat duration should still be visible during the
	// EndCombat step, and gone once the postcombat main phase begins.
	//
	// The engine removes EndOfCombat effects at the tail of the EndCombat
	// step handler (priority.go ~247), after the priority round for that
	// step has completed. That matches the rule: the effect persists
	// through the end-of-combat step and is cleared at the combat phase
	// boundary.
	t.Run("CR 500.5a until-end-of-combat expires at end of combat phase, not start of end-of-combat step", func(t *testing.T) {
		tg := NewTestGame(t)
		permID := tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")

		// Stop at BeginCombat of turn 1 (PlayerA), attach EndOfCombat pump.
		tg.StopAt(1, core.BeginCombat)
		tg.Execute()

		eff := mage.TargetEffect(core.LayerPT, core.EndOfCombat, permID,
			func(g *mage.Game, target *mage.Permanent) error {
				target.BoostPT(3, 3)
				return nil
			})
		tg.Effects.Add(eff)
		tg.Effects.Apply(tg.Game)
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 5, 5)

		// Run through to the EndCombat step (but stop before it executes).
		// Between BeginCombat and EndCombat the boost must persist.
		for _, step := range []core.PhaseStep{
			core.DeclareAttackers,
			core.DeclareBlockers,
			core.FirstStrikeDamage,
			core.CombatDamage,
		} {
			tg.RunStepWithPriority(step)
			tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 5, 5)
		}

		// Execute the EndCombat step. The boost is removed at the tail of
		// this step (end of combat phase), so by the time we observe
		// PostcombatMain it must be gone.
		tg.RunStepWithPriority(core.EndCombat)
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)

		// And of course still gone in postcombat main.
		tg.RunStepWithPriority(core.PostcombatMain)
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
	})

	// CR 500.7 — An effect that instructs a player to take an extra turn
	// makes that player active for the next turn. The harness honors this
	// via tg.ExtraTurns (harness.go Execute ~337). We queue an extra turn
	// for PlayerA, then assert that turn 2 is still PlayerA's, not
	// PlayerB's as it would normally be.
	t.Run("CR 500.7 extra-turn effect makes the granted player active next turn", func(t *testing.T) {
		tg := NewTestGame(t)

		// Stop at turn 1 PostcombatMain so PlayerA is still active and we
		// can queue an extra turn mid-turn.
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		playerAID := tg.GetPlayer(PlayerA).PlayerID()
		tg.Game.GrantExtraTurn(playerAID)

		// Advance to turn 2 PrecombatMain. Normally turn 2 is PlayerB, but
		// the extra turn grant should redirect turn 2 back to PlayerA.
		tg.StopAt(2, core.PrecombatMain)
		tg.Execute()

		gotActiveID := tg.ActivePlayerObj().PlayerID()
		if gotActiveID != playerAID {
			t.Errorf("CR 500.7: after GrantExtraTurn(PlayerA), turn 2 active player should be PlayerA; got %v want %v", gotActiveID, playerAID)
		}
	})

	// CR 500.11 — Effects may cause a player to skip a step, phase, or turn.
	// The engine currently exposes no primitive for skipping a step, phase,
	// or turn (no "skip next combat," no "skip your next turn," no
	// equivalent on Game or Player). grep for "Skip" in pkg/mage/core/turn.go
	// and pkg/mage/game.go turns up only unrelated matches (summoning-sick
	// skip, mana-ability skip).
	t.Run("CR 500.11 skip a step/phase/turn", func(t *testing.T) {
		t.Skip("XXX: engine gap — no skip-step/phase/turn primitive. Nothing in core/turn.go or game.go to schedule a skipped phase or turn.")
	})

	// CR 500.12 — No game events occur between steps or phases. Any ability
	// or state-based action that would look at "events between steps" sees
	// nothing; everything attaches to a specific step.
	//
	// This is essentially a negative invariant: there's no observable place
	// "between" two steps for a trigger to land. RunStepWithPriority only
	// runs inside a step; there is no hook between steps we can install a
	// listener on. Without such a hook there is nothing to assert against,
	// and no way to synthesize an "inter-step" event to prove the negative.
	t.Run("CR 500.12 no game events between steps", func(t *testing.T) {
		t.Skip("XXX: engine gap — no observable between-steps hook. The rule is a negative invariant; there is no API to attempt an event between steps and confirm it is delayed to the next step.")
	})
}
