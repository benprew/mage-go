package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
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

// replacementBase provides SourceID and GetDuration implementations for replacement effects.
type replacementBase struct {
	sourceID uuid.UUID
	duration Duration
}

func (r *replacementBase) SourceID() uuid.UUID  { return r.sourceID }
func (r *replacementBase) GetDuration() Duration { return r.duration }
