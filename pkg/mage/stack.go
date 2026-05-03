package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// StackObject represents something on the stack (spell or ability).
type StackObject struct {
	ID         uuid.UUID
	Card       Card // non-nil for spells
	Controller uuid.UUID
	SourceID   uuid.UUID // source permanent (for abilities)
	Effects    []Effect
	Targets    []uuid.UUID
	IsAbility  bool
	XValue      int // value of X for X-cost spells
	ModeChoice  int // chosen mode for modal spells (0-indexed)
	EventAmount int // amount from triggering event (e.g. damage dealt)
	// EventSourceID is the SourceID of the event that produced this
	// triggered ability (e.g. on EvtDamageDealt, the damager's ID).
	// Read by FuncEffect via Game.EventSourceID() during resolution.
	EventSourceID uuid.UUID

	// DamageDistribution carries per-target damage assignments for
	// divided-damage spells/abilities (CR 601.2d). The controller picks the
	// distribution at cast or activation time; the executor reads it back at
	// resolution. Map keys are the target IDs in StackObject.Targets; values
	// sum to the spell's total damage.
	DamageDistribution map[uuid.UUID]int

	// IsCopy marks this stack object as a copy of a spell (CR 707.10).
	// Copies of spells cease to exist when they resolve or are countered —
	// they do not enter any zone, are never put into a graveyard, and do
	// not become permanents. Set by Game.CopySpellOnStack.
	IsCopy bool

	// ModalTargets, when non-empty, holds the per-mode chosen targets for a
	// modal spell built with NewModalSpell. The slice at index ModeChoice
	// is the list of UUIDs the chosen mode's effects act on. Targets is
	// kept in sync (it points at ModalTargets[ModeChoice]) so existing
	// fizzle and target-still-legal logic works unchanged.
	ModalTargets [][]uuid.UUID

	// CastZone records the zone the spell was cast from (CR 601.2a).
	// Populated by the cast machinery: ZoneHand for the standard cast path,
	// or the explicit zone passed to CastCardFromZoneWithoutPaying /
	// CastCardFromZoneWithAlternateCost. Read at resolution time via
	// Game.ResolvingCastZone() so triggers can express "if you cast it from
	// your hand" / "from the graveyard" / etc.
	CastZone Zone

	// CastContext snapshots cast-time state used by effects whose Oracle
	// text references "as you cast this spell" (CR 608.2g). Populated when
	// a spell is pushed onto the stack and read during resolution via
	// Game.ResolvingCastContext(). Nil for activated/triggered abilities
	// and for spell stack objects whose cast path predates the snapshot
	// machinery; callers should treat nil as empty.
	CastContext *CastContext
}

// CastContext captures the cast-time snapshot mandated by CR 608.2g for
// effects that reference values "as you cast this spell". Fields are
// intentionally minimal — extend as new cards require.
//
// Populated by the cast pipeline immediately after additional costs have
// been paid (so reveal-style additional costs make it into the snapshot)
// and immediately before the StackObject is pushed.
type CastContext struct {
	// ControllerSubtypesAtCast records every subtype present on a permanent
	// the spell's controller controlled at the moment the spell went on the
	// stack. Used by cards like Draconic Roar ("controlled a Dragon as you
	// cast this spell"): a Dragon leaving between cast and resolution must
	// not flip the condition false.
	ControllerSubtypesAtCast map[string]bool

	// RevealedAtCast lists cards revealed as part of paying additional
	// costs for this spell (e.g. RevealFromHandCost). Used by cards like
	// Draconic Roar ("If you revealed a Dragon card... as you cast this
	// spell"). Card references are LKI: even if the card later changes
	// zones, this slice continues to point at the snapshotted Card.
	RevealedAtCast []Card
}

// HasControlledSubtypeAtCast is a small helper for the common
// "controlled an X as you cast this spell" predicate. Returns false if
// ctx is nil.
func (ctx *CastContext) HasControlledSubtypeAtCast(subtype string) bool {
	if ctx == nil || ctx.ControllerSubtypesAtCast == nil {
		return false
	}
	return ctx.ControllerSubtypesAtCast[subtype]
}

// RevealedSubtypeAtCast reports whether any card revealed as an additional
// cost for this spell had the given subtype. Returns false if ctx is nil.
func (ctx *CastContext) RevealedSubtypeAtCast(subtype string) bool {
	if ctx == nil {
		return false
	}
	for _, c := range ctx.RevealedAtCast {
		if c != nil && c.HasSubType(subtype) {
			return true
		}
	}
	return false
}

// Stack represents the game stack.
type Stack struct {
	objects []*StackObject
}

func NewStack() *Stack {
	return &Stack{}
}

func (s *Stack) Push(obj *StackObject) {
	s.objects = append(s.objects, obj)
}

func (s *Stack) Pop() *StackObject {
	if len(s.objects) == 0 {
		return nil
	}
	top := s.objects[len(s.objects)-1]
	s.objects = s.objects[:len(s.objects)-1]
	return top
}

func (s *Stack) IsEmpty() bool {
	return len(s.objects) == 0
}

func (s *Stack) Size() int {
	return len(s.objects)
}

func (s *Stack) Peek() *StackObject {
	if len(s.objects) == 0 {
		return nil
	}
	return s.objects[len(s.objects)-1]
}

// RemoveBySourceID removes a stack object matching the given source ID.
// The countered spell's card goes to its owner's graveyard.
func (s *Stack) RemoveBySourceID(sourceID uuid.UUID) *StackObject {
	for i, obj := range s.objects {
		if obj.SourceID == sourceID {
			s.objects = append(s.objects[:i], s.objects[i+1:]...)
			return obj
		}
	}
	return nil
}

// FindBySourceID finds a stack object by source ID without removing it.
func (s *Stack) FindBySourceID(sourceID uuid.UUID) *StackObject {
	for _, obj := range s.objects {
		if obj.SourceID == sourceID {
			return obj
		}
	}
	return nil
}

// Objects returns all objects on the stack for inspection.
func (s *Stack) Objects() []*StackObject {
	return s.objects
}
