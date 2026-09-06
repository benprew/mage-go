package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TurnSystem manages the turn counter, current phase/step, active player index,
// extra turns queue, turn step scheduling and skips (CR 500.7–500.11), and
// per-permanent untap skipping.
type TurnSystem struct {
	turn          int
	step          PhaseStep
	activePlayer  int // index into players
	extraTurns    []uuid.UUID
	schedule      *TurnSchedule
	skipNextUntap map[uuid.UUID]int
}

// NewTurnSystem creates an initialized TurnSystem starting at turn 1.
func NewTurnSystem() TurnSystem {
	return TurnSystem{
		turn:          1,
		schedule:      newTurnSchedule(),
		skipNextUntap: make(map[uuid.UUID]int),
	}
}

// Clone creates an independent deep copy of TurnSystem.
func (ts TurnSystem) Clone() TurnSystem {
	clone := TurnSystem{
		turn:          ts.turn,
		step:          ts.step,
		activePlayer:  ts.activePlayer,
		skipNextUntap: cloneUUIDMap(ts.skipNextUntap),
	}
	if len(ts.extraTurns) > 0 {
		clone.extraTurns = make([]uuid.UUID, len(ts.extraTurns))
		copy(clone.extraTurns, ts.extraTurns)
	}
	if ts.schedule != nil {
		clone.schedule = ts.schedule.Clone()
	}
	return clone
}

// Turn returns the current turn number (1-indexed).
func (ts *TurnSystem) Turn() int {
	return ts.turn
}

// SetTurn overrides the current turn number.
func (ts *TurnSystem) SetTurn(n int) {
	ts.turn = n
}

// IncrementTurn advances the turn number by 1.
func (ts *TurnSystem) IncrementTurn() {
	ts.turn++
}

// Step returns the current phase or step.
func (ts *TurnSystem) Step() PhaseStep {
	return ts.step
}

// SetStep sets the current phase or step.
func (ts *TurnSystem) SetStep(s PhaseStep) {
	ts.step = s
}

// ActivePlayerIndex returns the index of the active player in Game.players.
func (ts *TurnSystem) ActivePlayerIndex() int {
	return ts.activePlayer
}

// SetActivePlayerIndex sets the active player index.
func (ts *TurnSystem) SetActivePlayerIndex(idx int) {
	ts.activePlayer = idx
}

// ExtraTurns returns the slice of player IDs scheduled for extra turns.
func (ts *TurnSystem) ExtraTurns() []uuid.UUID {
	return ts.extraTurns
}

// GrantExtraTurn appends an extra turn for playerID to the schedule.
func (ts *TurnSystem) GrantExtraTurn(playerID uuid.UUID) {
	ts.extraTurns = append(ts.extraTurns, playerID)
}

// PopExtraTurn dequeues the next extra turn recipient, if any.
func (ts *TurnSystem) PopExtraTurn() (uuid.UUID, bool) {
	if len(ts.extraTurns) == 0 {
		return uuid.Nil, false
	}
	next := ts.extraTurns[0]
	ts.extraTurns = ts.extraTurns[1:]
	return next, true
}

// HasExtraTurns reports whether any extra turns are queued.
func (ts *TurnSystem) HasExtraTurns() bool {
	return len(ts.extraTurns) > 0
}

// Schedule returns the TurnSchedule pointer, initializing it if nil.
func (ts *TurnSystem) Schedule() *TurnSchedule {
	if ts.schedule == nil {
		ts.schedule = newTurnSchedule()
	}
	return ts.schedule
}

// SetSchedule replaces the active TurnSchedule.
func (ts *TurnSystem) SetSchedule(s *TurnSchedule) {
	ts.schedule = s
}

// AppendExtraStep queues step to run at the end of the remaining schedule
// for the current turn. CR 500.9 / 500.10.
func (ts *TurnSystem) AppendExtraStep(step PhaseStep) {
	s := ts.Schedule()
	s.Remaining = append(s.Remaining, step)
}

// InsertStepAfter inserts step into the remaining schedule directly after
// the first occurrence of anchor. If anchor is not in the remaining
// schedule, step is appended. CR 500.9.
func (ts *TurnSystem) InsertStepAfter(anchor, step PhaseStep) {
	s := ts.Schedule()
	for i, st := range s.Remaining {
		if st == anchor {
			rest := append([]PhaseStep{step}, s.Remaining[i+1:]...)
			s.Remaining = append(s.Remaining[:i+1], rest...)
			return
		}
	}
	ts.AppendExtraStep(step)
}

// SkipNextOccurrenceOfStep marks one upcoming occurrence of step to be skipped.
func (ts *TurnSystem) SkipNextOccurrenceOfStep(step PhaseStep) {
	ts.Schedule().SkipNextStep[step]++
}

// SkipNextCombatPhase queues skips for every step in the combat phase.
func (ts *TurnSystem) SkipNextCombatPhase() {
	s := ts.Schedule()
	for _, step := range []PhaseStep{
		BeginCombat,
		DeclareAttackers,
		DeclareBlockers,
		FirstStrikeDamage,
		CombatDamage,
		EndCombat,
	} {
		s.SkipNextStep[step]++
	}
}

// SkipNextTurnFor marks one upcoming turn of playerID as skipped.
func (ts *TurnSystem) SkipNextTurnFor(playerID uuid.UUID) {
	ts.Schedule().SkipNextTurnFor[playerID]++
}

// ConsumeTurnSkip consumes one pending turn skip for playerID if present.
func (ts *TurnSystem) ConsumeTurnSkip(playerID uuid.UUID) bool {
	return ts.Schedule().consumeTurnSkip(playerID)
}

// BuildNextTurn populates Remaining with the canonical step list for a new turn.
func (ts *TurnSystem) BuildNextTurn() {
	ts.Schedule().buildNextTurn()
}

// PopNextStep returns the next step to execute, or (0, false) if the turn is done.
func (ts *TurnSystem) PopNextStep() (PhaseStep, bool) {
	return ts.Schedule().popNextStep()
}

// SkipNextUntap causes the specified permanent to skip its next untap attempt.
func (ts *TurnSystem) SkipNextUntap(permanentID uuid.UUID) {
	if ts.skipNextUntap == nil {
		ts.skipNextUntap = make(map[uuid.UUID]int)
	}
	ts.skipNextUntap[permanentID]++
}

// RemoveSkipNextUntap removes any pending untap skips for a permanent that leaves the battlefield.
func (ts *TurnSystem) RemoveSkipNextUntap(permanentID uuid.UUID) {
	delete(ts.skipNextUntap, permanentID)
}

// ConsumeSkipNextUntap decrements/clears the skip count if active and returns true if untap was skipped.
func (ts *TurnSystem) ConsumeSkipNextUntap(permanentID uuid.UUID) bool {
	if remaining := ts.skipNextUntap[permanentID]; remaining > 0 {
		if remaining == 1 {
			delete(ts.skipNextUntap, permanentID)
		} else {
			ts.skipNextUntap[permanentID] = remaining - 1
		}
		return true
	}
	return false
}

// HasSkipNextUntap reports whether the permanent has an active untap skip.
func (ts *TurnSystem) HasSkipNextUntap(permanentID uuid.UUID) bool {
	return ts.skipNextUntap[permanentID] > 0
}
