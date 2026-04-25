// Package gametest: turn-structure tests for the beginning phase.
// Covers CR 502 (untap), 503 (upkeep), 503.2 (multiple upkeep steps),
// 504 (draw), and 103.8a (first-turn draw rule).
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// TestTurnStructureBeginning covers CR 502 (untap), 503 (upkeep), 504 (draw),
// and 103.8a (first-turn draw rule).
//
// Turn/active-player layout in this harness:
//   Turn 1 → PlayerA active
//   Turn 2 → PlayerB active
//   Turn 3 → PlayerA active

// CR 502.3 — At the beginning of the untap step, the active player
// determines which of their permanents untap, then they untap simultaneously.
// We stop at turn 3 Untap (before it runs), tap PlayerA's creature, then
// run Untap and assert the creature untapped.
func TestTurnStructureBeginning_Untap_ActivePlayerUntaps(t *testing.T) {
	tg := NewTestGame(t)
	permID := tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	perm := tg.FindPermanent(permID)
	if perm == nil {
		t.Fatalf("permanent not found after AddCard")
	}

	// Turn 3 = PlayerA active. Stop before Untap runs, then tap.
	tg.StopAt(3, core.Untap)
	tg.Execute()
	perm.Tapped = true

	tg.RunStepWithPriority(core.Untap)
	tg.AssertTapped(PlayerA, "Grizzly Bears", false)
}

// CR 502.3 (negative) — Only the active player's permanents untap. A
// non-active player's tapped permanents remain tapped through the active
// player's untap step. We tap PlayerA's creature *after* turn 1 untap has
// already run (by stopping at turn 2 Untap, which is before PlayerB's
// untap step executes), then drive the rest of turn 2 manually.
func TestTurnStructureBeginning_Untap_NonActivePlayerDoesNotUntap(t *testing.T) {
	tg := NewTestGame(t)
	permID := tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	perm := tg.FindPermanent(permID)
	if perm == nil {
		t.Fatalf("permanent not found after AddCard")
	}

	// Advance to turn 2 Untap (returns before Untap runs). Turn 2 = PlayerB.
	tg.StopAt(2, core.Untap)
	tg.Execute()

	// Tap PlayerA's (non-active) creature now.
	perm.Tapped = true

	// Run PlayerB's Untap step. PlayerA's creature must NOT untap.
	tg.RunStepWithPriority(core.Untap)
	tg.AssertTapped(PlayerA, "Grizzly Bears", true)
}

// CR 502.3 — After being tapped during the non-active player's turn, the
// creature untaps on its own controller's next untap step (turn 3). This
// confirms the negative test wasn't caused by permanently stuck state.
func TestTurnStructureBeginning_Untap_UntapsOnOwnTurnAfterStayingTapped(t *testing.T) {
	tg := NewTestGame(t)
	permID := tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	perm := tg.FindPermanent(permID)
	if perm == nil {
		t.Fatalf("permanent not found after AddCard")
	}

	// Stop at turn 2 Untap (PlayerB), tap PlayerA's creature.
	tg.StopAt(2, core.Untap)
	tg.Execute()
	perm.Tapped = true

	// Run the rest of turn 2 and all steps up to turn 3 PrecombatMain.
	tg.StopAt(3, core.PrecombatMain)
	tg.Execute()

	tg.AssertTapped(PlayerA, "Grizzly Bears", false)
}

// CR 502.4 — No player receives priority during the untap step. Abilities
// that trigger during the untap step are held and first go on the stack at
// the beginning of the upkeep step. We use an EvtBecameUntapped trigger
// that gains 1 life for its controller; we observe that:
//
//	(a) stopping at the start of upkeep (before priority) shows life
//	    unchanged — the trigger was not able to resolve during untap.
//	(b) stopping after upkeep shows the trigger resolved.
func TestTurnStructureBeginning_Untap_NoPriority_TriggerHeldUntilUpkeep(t *testing.T) {
	name := "TSB Untap Life Gainer"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewArtifact(name, "{2}",
				mage.WithAbility(mage.NewTriggered(
					core.EvtBecameUntapped,
					false,
					mage.GainLife(1),
				).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == sourceID
				})),
			)
		})
	}

	// (a) Stop at turn 3 Untap (before untap runs), then tap the artifact
	// so that untap this turn is the first untap event since it became
	// tapped.
	tg := NewTestGame(t)
	permID := tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	perm := tg.FindPermanent(permID)
	if perm == nil {
		t.Fatalf("permanent not found after AddCard")
	}
	tg.StopAt(3, core.Untap)
	tg.Execute()
	perm.Tapped = true
	preLife := tg.GetPlayer(PlayerA).Life()

	// Run Untap. No priority is granted, so the triggered ability
	// cannot resolve yet — life is unchanged.
	tg.RunStepWithPriority(core.Untap)
	if tg.GetPlayer(PlayerA).Life() != preLife {
		t.Errorf("CR 502.4: expected no life change during untap step (no priority); pre=%d got=%d",
			preLife, tg.GetPlayer(PlayerA).Life())
	}

	// (b) Now run Upkeep. The held trigger goes on the stack and
	// resolves, gaining 1 life.
	tg.RunStepWithPriority(core.Upkeep)
	if tg.GetPlayer(PlayerA).Life() != preLife+1 {
		t.Errorf("CR 502.4: expected +1 life after upkeep resolves held trigger; pre=%d got=%d",
			preLife, tg.GetPlayer(PlayerA).Life())
	}
}

// CR 503.1a — Abilities that trigger at the beginning of the upkeep step
// go on the stack and resolve before the draw step begins. We observe
// ordering by having an upkeep trigger draw a card: if the trigger
// resolves before the normal draw, the active player gains 2 cards this
// turn (1 from the trigger + 1 from the draw step).
func TestTurnStructureBeginning_Upkeep_TriggerResolvesBeforeDraw(t *testing.T) {
	name := "TSB Upkeep Draw Trigger"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewArtifact(name, "{2}",
				mage.WithAbility(mage.NewTriggered(
					core.EvtUpkeep,
					false,
					mage.DrawCardsActivePlayer(mage.Fixed(1)),
				).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, controllerID uuid.UUID) bool {
					return evt.PlayerID == controllerID
				})),
			)
		})
	}

	// Measure hand size at the start of turn 3 upkeep (before upkeep
	// trigger and before draw step).
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(3, core.Upkeep)
	tg.Execute()
	preHand := len(tg.GetPlayer(PlayerA).Hand())

	// Now run to precombat main of turn 3: upkeep trigger + draw step
	// have both happened.
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg2.StopAt(3, core.PrecombatMain)
	tg2.Execute()
	postHand := len(tg2.GetPlayer(PlayerA).Hand())

	if postHand-preHand != 2 {
		t.Errorf("CR 503.1a: expected hand to grow by 2 (trigger draw + draw step), got +%d (pre=%d post=%d)",
			postHand-preHand, preHand, postHand)
	}
}

// TestTurnStructureBeginning_Upkeep_MultipleUpkeepSteps verifies CR 503.2:
// a turn can contain more than one upkeep step. An effect that adds an extra
// upkeep step (e.g. Paradox Haze) inserts an additional upkeep step into the
// current turn, and "at the beginning of upkeep" triggers fire for each one.
//
// Engine primitive: Game.InsertStepAfter(anchor, step) on the per-turn
// TurnSchedule, exposed via GameMutator so card effects can call it.
func TestTurnStructureBeginning_Upkeep_MultipleUpkeepSteps(t *testing.T) {
	// A pinger that gains 1 life at the beginning of its controller's
	// upkeep. With a single upkeep step the controller gains 1 life per
	// turn; with an extra upkeep step inserted, 2 life per turn.
	pinger := "CR 503.2 Upkeep Lifegainer"
	if !mage.CardRegistered(pinger) {
		mage.Register(pinger, func() mage.Card {
			return mage.NewArtifact(pinger, "{0}",
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

	// Baseline: with a single upkeep step, PlayerA gains 1 life on their
	// upkeep of turn 3 (turns 1 and 3 are PlayerA's). Check the delta
	// across turn 3's upkeep specifically.
	tgBase := NewTestGame(t)
	tgBase.AddCard(core.ZoneBattlefield, PlayerA, pinger)
	tgBase.StopAt(3, core.Upkeep)
	tgBase.Execute()
	preBase := tgBase.GetPlayer(PlayerA).Life()
	tgBase.StopAt(3, core.Draw)
	tgBase.Execute()
	postBase := tgBase.GetPlayer(PlayerA).Life()
	if postBase-preBase != 1 {
		t.Fatalf("baseline: expected +1 life across single upkeep, got +%d (pre=%d post=%d)",
			postBase-preBase, preBase, postBase)
	}

	// CR 503.2: insert an extra upkeep step into turn 3 immediately after
	// the canonical upkeep. Stop before upkeep, mutate the schedule, then
	// run through the remaining steps in the turn.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, pinger)
	tg.StopAt(3, core.Upkeep)
	tg.Execute()
	pre := tg.GetPlayer(PlayerA).Life()

	// The stop re-queues Upkeep at the head of Remaining. Insert an extra
	// Upkeep right after it so the turn runs: Upkeep, Upkeep, Draw, ...
	tg.InsertStepAfter(core.Upkeep, core.Upkeep)

	// Drive turn 3 forward: run Upkeep (original), then Upkeep (extra),
	// then stop before Draw so we can measure.
	tg.RunStepWithPriority(core.Upkeep)
	step2, ok := tg.Schedule.PopNextStep()
	if !ok || step2 != core.Upkeep {
		t.Fatalf("CR 503.2: expected a second Upkeep step in Remaining, got step=%v ok=%v", step2, ok)
	}
	tg.Step = core.Upkeep
	tg.RunStepWithPriority(core.Upkeep)

	post := tg.GetPlayer(PlayerA).Life()
	if post-pre != 2 {
		t.Errorf("CR 503.2: expected +2 life across two upkeep steps, got +%d (pre=%d post=%d)",
			post-pre, pre, post)
	}
}

// CR 504.1 — First, the active player draws a card during their draw step.
func TestTurnStructureBeginning_Draw_ActivePlayerDraws(t *testing.T) {
	// Before draw step of turn 3 (PlayerA active).
	tg := NewTestGame(t)
	tg.StopAt(3, core.Draw)
	tg.Execute()
	preHand := len(tg.GetPlayer(PlayerA).Hand())

	// After draw step of turn 3.
	tg2 := NewTestGame(t)
	tg2.StopAt(3, core.PrecombatMain)
	tg2.Execute()
	postHand := len(tg2.GetPlayer(PlayerA).Hand())

	if postHand-preHand != 1 {
		t.Errorf("CR 504.1: expected active player to draw 1 during draw step; pre=%d post=%d",
			preHand, postHand)
	}
}

// CR 504 — Only the active player draws during their draw step. The
// non-active player's hand size is unchanged across that step.
func TestTurnStructureBeginning_Draw_NonActivePlayerDoesNotDraw(t *testing.T) {
	// Turn 3 = PlayerA active, PlayerB non-active.
	tg := NewTestGame(t)
	tg.StopAt(3, core.Draw)
	tg.Execute()
	preB := len(tg.GetPlayer(PlayerB).Hand())

	tg2 := NewTestGame(t)
	tg2.StopAt(3, core.PrecombatMain)
	tg2.Execute()
	postB := len(tg2.GetPlayer(PlayerB).Hand())

	if preB != postB {
		t.Errorf("CR 504: non-active player should not draw during active player's draw step; pre=%d post=%d",
			preB, postB)
	}
}

// CR 103.8a — In a two-player game, the player who plays first skips the
// draw step of their first turn.
func TestTurnStructureBeginning_Draw_FirstPlayerSkipsTurnOneDraw(t *testing.T) {
	// Before the turn-1 draw step.
	tg := NewTestGame(t)
	tg.StopAt(1, core.Draw)
	tg.Execute()
	preHand := len(tg.GetPlayer(PlayerA).Hand())

	// After the turn-1 draw step.
	tg2 := NewTestGame(t)
	tg2.StopAt(1, core.PrecombatMain)
	tg2.Execute()
	postHand := len(tg2.GetPlayer(PlayerA).Hand())

	if postHand != preHand {
		t.Errorf("CR 103.8a: starting player should not draw on turn 1; pre=%d post=%d",
			preHand, postHand)
	}
}
