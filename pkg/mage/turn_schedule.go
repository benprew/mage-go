package mage

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TurnSchedule tracks the remaining steps of the current turn and any
// pending skips for upcoming steps, phases, and turns. It implements the
// CR 500.7 / 500.8 / 500.9 / 500.11 "extra/skipped step/phase/turn" primitives.
//
// The driver (Run / RunTurn) and the gametest harness read from this
// schedule instead of iterating core.AllSteps() directly, so that cards and
// effects can mutate the turn shape.
type TurnSchedule struct {
	// Remaining is the ordered list of steps still to execute in the
	// current turn. The driver pops from the front. Extra steps inserted
	// via AppendExtraStep / InsertStepAfter land here.
	Remaining []core.PhaseStep

	// SkipNextStep[step] is the count of upcoming occurrences of step to
	// skip. Decremented each time a step is popped. A non-zero entry
	// causes the driver to drop that step from the schedule (as the turn
	// is built) or to drop it on the current turn before it runs.
	SkipNextStep map[core.PhaseStep]int

	// SkipNextTurnFor[playerID] is the number of upcoming turns to skip
	// for that player. Decremented when a player would become the active
	// player for a new turn.
	SkipNextTurnFor map[uuid.UUID]int
}

// NewTurnSchedule constructs an empty TurnSchedule. Normally callers don't
// need this — Game.Schedule is initialized by NewGame — but it's exported
// for harnesses that may need to reset it.
func NewTurnSchedule() *TurnSchedule {
	return newTurnSchedule()
}

// ConsumeTurnSkip exposes consumeTurnSkip for external harnesses.
func (ts *TurnSchedule) ConsumeTurnSkip(playerID uuid.UUID) bool {
	return ts.consumeTurnSkip(playerID)
}

// BuildNextTurn exposes buildNextTurn.
func (ts *TurnSchedule) BuildNextTurn() { ts.buildNextTurn() }

// PopNextStep exposes popNextStep.
func (ts *TurnSchedule) PopNextStep() (core.PhaseStep, bool) { return ts.popNextStep() }

func newTurnSchedule() *TurnSchedule {
	return &TurnSchedule{
		SkipNextStep:    make(map[core.PhaseStep]int),
		SkipNextTurnFor: make(map[uuid.UUID]int),
	}
}

// buildNextTurn populates Remaining with the canonical step list, applying
// any pending one-shot step skips. Called by the driver at the start of a
// turn.
func (ts *TurnSchedule) buildNextTurn() {
	canonical := core.AllSteps()
	out := make([]core.PhaseStep, 0, len(canonical))
	for _, s := range canonical {
		if ts.SkipNextStep[s] > 0 {
			ts.SkipNextStep[s]--
			continue
		}
		out = append(out, s)
	}
	ts.Remaining = out
}

// popNextStep returns the next step to execute, or (_, false) if the turn
// is done.
func (ts *TurnSchedule) popNextStep() (core.PhaseStep, bool) {
	for len(ts.Remaining) > 0 {
		step := ts.Remaining[0]
		ts.Remaining = ts.Remaining[1:]
		if ts.SkipNextStep[step] > 0 {
			ts.SkipNextStep[step]--
			continue
		}
		return step, true
	}
	return 0, false
}

// AppendExtraStep queues step to run at the end of the remaining schedule
// for the current turn. CR 500.9 / 500.10.
func (g *Game) AppendExtraStep(step core.PhaseStep) {
	g.Schedule.Remaining = append(g.Schedule.Remaining, step)
}

// InsertStepAfter inserts step into the remaining schedule directly after
// the first occurrence of anchor. If anchor is not in the remaining
// schedule, step is appended. CR 500.9.
func (g *Game) InsertStepAfter(anchor, step core.PhaseStep) {
	for i, s := range g.Schedule.Remaining {
		if s == anchor {
			rest := append([]core.PhaseStep{step}, g.Schedule.Remaining[i+1:]...)
			g.Schedule.Remaining = append(g.Schedule.Remaining[:i+1], rest...)
			return
		}
	}
	g.AppendExtraStep(step)
}

// SkipNextOccurrenceOfStep marks one upcoming occurrence of step to be
// skipped. If the step is still in the current turn's Remaining schedule,
// the next pop will drop it; otherwise it applies to a future turn. CR 500.11.
func (g *Game) SkipNextOccurrenceOfStep(step core.PhaseStep) {
	g.Schedule.SkipNextStep[step]++
}

// SkipNextCombatPhase queues skips for every step in the combat phase
// (BeginCombat through EndCombat) for the next turn that reaches them.
// Used for effects like "skip your next combat phase". CR 500.11, 506.1.
func (g *Game) SkipNextCombatPhase() {
	for _, s := range []core.PhaseStep{
		core.BeginCombat,
		core.DeclareAttackers,
		core.DeclareBlockers,
		core.FirstStrikeDamage,
		core.CombatDamage,
		core.EndCombat,
	} {
		g.Schedule.SkipNextStep[s]++
	}
}

// SkipNextTurnFor marks one upcoming turn of playerID as skipped. The
// driver detects this at turn-start and proceeds to the following player /
// extra turn. CR 500.11, 500.7.
func (g *Game) SkipNextTurnFor(playerID uuid.UUID) {
	g.Schedule.SkipNextTurnFor[playerID]++
}

// consumeTurnSkip returns true if the active player's next turn should be
// skipped, consuming one pending skip.
func (ts *TurnSchedule) consumeTurnSkip(playerID uuid.UUID) bool {
	if ts.SkipNextTurnFor[playerID] > 0 {
		ts.SkipNextTurnFor[playerID]--
		return true
	}
	return false
}
