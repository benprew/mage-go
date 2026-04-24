// Package gametest: turn-structure tests for the end step.
// Covers CR 512 (ending phase) and CR 513 (end step): 513.1 (at-beginning-of-
// end-step triggers) and 513.2 (permanents entering during the end step do not
// fire their own end-step triggers that turn).
package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestEndStep covers CR 513 (end step) sub-rules. See CR 500.6 for the
// general "At the beginning of X" trigger rule.
func TestEndStep(t *testing.T) {

	// CR 513.1 / 500.6 — "At the beginning of the end step" triggered abilities
	// go on the stack at the start of the end step and resolve before the
	// cleanup step begins.
	t.Run("CR 513.1 at beginning of end step trigger fires", func(t *testing.T) {
		name := "EndStep Life Gainer"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewArtifact(name, "{2}",
					mage.WithAbility(mage.NewTriggered(
						core.EvtEndStep,
						false,
						mage.GainLife(5),
					).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, controllerID uuid.UUID) bool {
						return evt.PlayerID == controllerID
					})),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.StopAt(2, core.Upkeep)
		tg.Execute()

		// Starting life 20 + 5 from turn 1 end step trigger = 25.
		tg.AssertLife(PlayerA, 25)
	})

	// CR 513.2 — "Any abilities that triggered during the turn and are
	// waiting to be put onto the stack are put onto the stack during the end
	// step." In particular, a permanent that enters the battlefield during
	// the end step does NOT have its own "at the beginning of the end step"
	// trigger fire that turn — the EvtEndStep event has already been
	// processed. The trigger fires at the next turn's end step instead.
	t.Run("CR 513.2 permanent entering during end step does not trigger until next turn", func(t *testing.T) {
		name := "ECU2 513.2 End Step Drainer"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewArtifact(name, "{2}",
					mage.WithAbility(mage.NewTriggered(
						core.EvtEndStep,
						false,
						mage.GainLife(3),
					).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, controllerID uuid.UUID) bool {
						return evt.PlayerID == controllerID
					})),
				)
			})
		}

		tg := NewTestGame(t)
		tg.StopAt(1, core.EndStep)
		tg.Execute()
		// At this point turn 1's EndStep has been selected but not yet run
		// (Execute returns before the stop step is executed).

		// Install a priority handler that, the first time a player gets
		// priority during turn 1's end step, puts the trigger-bearing
		// artifact onto the battlefield under PlayerA's control. The
		// EvtEndStep event has already been fired by doEndStepActions by
		// this point, so the new permanent should not see that event.
		injected := false
		aID := tg.getPlayerID(PlayerA)
		tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
			if !injected && g.Turn == 1 && g.Step == core.EndStep {
				card, err := mage.CreateCard(name)
				if err != nil {
					t.Fatalf("CreateCard: %v", err)
				}
				card.SetOwner(aID)
				g.PutOnBattlefield(card, aID)
				injected = true
			}
			return mage.PriorityAction{Type: mage.PriorityPass}
		}

		// Run turn 1's EndStep (event already fired; our handler drops the
		// artifact in during the priority round) and Cleanup.
		tg.Step = core.EndStep
		tg.RunStepWithPriority(core.EndStep)
		tg.Step = core.Cleanup
		tg.RunStepWithPriority(core.Cleanup)

		if !injected {
			t.Fatalf("CR 513.2 setup: artifact was never injected during turn 1 end step")
		}
		// No end-step trigger should have resolved this turn.
		tg.AssertLife(PlayerA, 20)

		// Advance to turn 2 and run through its end step.
		tg.ActivePlayer = (tg.ActivePlayer + 1) % len(tg.Players)
		tg.Turn++
		for _, step := range core.AllSteps() {
			tg.Step = step
			tg.RunStepWithPriority(step)
			if step == core.EndStep {
				break
			}
		}
		// Turn 2's end step: active player is PlayerB; the trigger is
		// controlled by PlayerA and conditioned on evt.PlayerID ==
		// controllerID. So it does NOT fire in PlayerB's end step either.
		tg.AssertLife(PlayerA, 20)

		// Continue to turn 3 (PlayerA active) and run through its end step.
		// Finish turn 2's Cleanup first.
		tg.Step = core.Cleanup
		tg.RunStepWithPriority(core.Cleanup)
		tg.ActivePlayer = (tg.ActivePlayer + 1) % len(tg.Players)
		tg.Turn++
		for _, step := range core.AllSteps() {
			tg.Step = step
			tg.RunStepWithPriority(step)
			if step == core.EndStep {
				break
			}
		}
		// Now PlayerA's end step has fired with the artifact present: +3 life.
		tg.AssertLife(PlayerA, 23)
	})
}
