package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Action represents a game mutation flowing through the replacement pipeline.
// The pipeline transforms or replaces actions before they execute.
type Action interface {
	ActionSource() uuid.UUID
}

// ReplacementEffect intercepts and modifies Actions before they resolve.
// Each replacement can match specific action types and either modify or
// fully replace them (returning nil means the action is fully prevented).
type ReplacementEffect interface {
	// Matches returns true if this replacement applies to the given action.
	Matches(Action, GameReader) bool
	// Replace transforms the action. Return nil to fully prevent it.
	// The returned action may be a different type (e.g. redirect damage).
	Replace(Action, *Game) Action
	// SourceID returns the permanent/spell that created this replacement.
	SourceID() uuid.UUID
	// IsActive returns true if this replacement is still valid.
	IsActive(GameReader) bool
	// GetDuration returns when this replacement expires.
	GetDuration() Duration
	// Clone returns a deep copy of this replacement effect (for game cloning).
	Clone() ReplacementEffect
}

// ReplacementPriority defines priority stages under CR 616.1.
type ReplacementPriority int

const (
	ReplacementPrioritySelf    ReplacementPriority = 100 // CR 616.1a: Self-replacement effects (CR 614.15)
	ReplacementPriorityControl ReplacementPriority = 200 // CR 616.1b: Control-changing effects on entry
	ReplacementPriorityCopy    ReplacementPriority = 300 // CR 616.1c: Copy effects on entry
	ReplacementPriorityFace    ReplacementPriority = 400 // CR 616.1d: Face-down / back face on entry
	ReplacementPriorityDefault ReplacementPriority = 500 // CR 616.1e: Other replacement or prevention effects
)

// ReplacementPriorityProvider allows replacement effects to declare their CR 616.1 priority stage.
type ReplacementPriorityProvider interface {
	ReplacementPriority() ReplacementPriority
}

// replacementBase provides SourceID and GetDuration implementations for replacement effects.
type replacementBase struct {
	sourceID uuid.UUID
	duration Duration
	priority ReplacementPriority
}

func (r *replacementBase) SourceID() uuid.UUID   { return r.sourceID }
func (r *replacementBase) GetDuration() Duration { return r.duration }
func (r *replacementBase) ReplacementPriority() ReplacementPriority {
	if r.priority == 0 {
		return ReplacementPriorityDefault
	}
	return r.priority
}
func (r *replacementBase) SetReplacementPriority(p ReplacementPriority) {
	r.priority = p
}
