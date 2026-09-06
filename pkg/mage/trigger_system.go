package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// DelayedTrigger represents a one-shot triggered ability that fires when
// a specific event occurs (e.g., "destroy this creature at end of turn").
type DelayedTrigger struct {
	EventType     EventType
	TargetID      uuid.UUID
	Effects       []Effect
	SourceID      uuid.UUID
	Controller    uuid.UUID
	MatchEventID  uuid.UUID // if set, only fire when evt.SourceID matches
	MatchPlayerID uuid.UUID // if set, only fire when evt.PlayerID matches
	MatchTargetID uuid.UUID // if set, only fire when evt.TargetID matches
	MatchFromZone Zone      // for EvtZoneChange: ZoneAny to skip the from check
	MatchToZone   Zone      // for EvtZoneChange: ZoneAny to skip the to check
	MatchFlag     bool      // if true, only fire when evt.Flag is true (e.g. combat damage)
	Persistent    bool      // if true, trigger is not consumed after firing
}

type pendingTrigger struct {
	ability    TriggeredAbility
	event      *GameEvent
	sourceID   uuid.UUID
	controller uuid.UUID
}

type stateTriggerKey struct {
	sourceID  uuid.UUID
	abilityID uuid.UUID
}

// TriggerSystem encapsulates pending triggers, delayed triggers, and armed state triggers.
type TriggerSystem struct {
	pending    []*pendingTrigger
	delayed    []*DelayedTrigger
	armedState map[stateTriggerKey]bool
}

// NewTriggerSystem returns an initialized TriggerSystem.
func NewTriggerSystem() TriggerSystem {
	return TriggerSystem{
		armedState: make(map[stateTriggerKey]bool),
	}
}

// RegisterDelayed registers a delayed trigger.
func (ts *TriggerSystem) RegisterDelayed(dt *DelayedTrigger) {
	if dt != nil {
		ts.delayed = append(ts.delayed, dt)
	}
}

// Delayed returns the active delayed triggers.
func (ts *TriggerSystem) Delayed() []*DelayedTrigger {
	return ts.delayed
}

// SetDelayed replaces the delayed triggers slice.
func (ts *TriggerSystem) SetDelayed(delayed []*DelayedTrigger) {
	ts.delayed = delayed
}

// AddPending adds a pending trigger.
func (ts *TriggerSystem) AddPending(pt *pendingTrigger) {
	if pt != nil {
		ts.pending = append(ts.pending, pt)
	}
}

// Pending returns the active pending triggers.
func (ts *TriggerSystem) Pending() []*pendingTrigger {
	return ts.pending
}

// SetPending sets the pending triggers slice.
func (ts *TriggerSystem) SetPending(pending []*pendingTrigger) {
	ts.pending = pending
}

// ClearPending clears all pending triggers.
func (ts *TriggerSystem) ClearPending() {
	ts.pending = nil
}

// ArmedState returns the map of armed state triggers.
func (ts *TriggerSystem) ArmedState() map[stateTriggerKey]bool {
	return ts.armedState
}

// IsStateTriggerArmed reports whether the given state trigger is already armed.
func (ts *TriggerSystem) IsStateTriggerArmed(key stateTriggerKey) bool {
	return ts.armedState[key]
}

// ArmStateTrigger marks a state trigger as armed.
func (ts *TriggerSystem) ArmStateTrigger(key stateTriggerKey) {
	if ts.armedState == nil {
		ts.armedState = make(map[stateTriggerKey]bool)
	}
	ts.armedState[key] = true
}

// DisarmStateTrigger removes an armed state trigger.
func (ts *TriggerSystem) DisarmStateTrigger(key stateTriggerKey) {
	if ts.armedState != nil {
		delete(ts.armedState, key)
	}
}

// ClearUnseenArmed removes armed state triggers whose sources are no longer seen.
func (ts *TriggerSystem) ClearUnseenArmed(seen map[stateTriggerKey]bool) {
	for key := range ts.armedState {
		if !seen[key] {
			delete(ts.armedState, key)
		}
	}
}

// ClearEndOfTurn removes persistent (turn-scoped) delayed triggers.
func (ts *TriggerSystem) ClearEndOfTurn() {
	kept := ts.delayed[:0]
	for _, dt := range ts.delayed {
		if !dt.Persistent {
			kept = append(kept, dt)
		}
	}
	ts.delayed = kept
}

// Clone creates an independent deep copy of TriggerSystem.
func (ts *TriggerSystem) Clone() TriggerSystem {
	var clonedPending []*pendingTrigger
	if len(ts.pending) > 0 {
		clonedPending = make([]*pendingTrigger, len(ts.pending))
		for i, pt := range ts.pending {
			clone := *pt
			clonedPending[i] = &clone
		}
	}

	var clonedDelayed []*DelayedTrigger
	if len(ts.delayed) > 0 {
		clonedDelayed = make([]*DelayedTrigger, len(ts.delayed))
		for i, dt := range ts.delayed {
			clone := *dt
			if len(dt.Effects) > 0 {
				clone.Effects = append([]Effect(nil), dt.Effects...)
			}
			clonedDelayed[i] = &clone
		}
	}

	var clonedArmed map[stateTriggerKey]bool
	if len(ts.armedState) > 0 {
		clonedArmed = make(map[stateTriggerKey]bool, len(ts.armedState))
		maps.Copy(clonedArmed, ts.armedState)
	}

	return TriggerSystem{
		pending:    clonedPending,
		delayed:    clonedDelayed,
		armedState: clonedArmed,
	}
}
