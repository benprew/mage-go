package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ResolutionState encapsulates transient context for spells and abilities
// that are currently resolving from the stack or being processed during costs.
type ResolutionState struct {
	// X value for the currently resolving spell or activation
	currentX int

	// Chosen mode for the currently resolving modal spell (0-indexed)
	currentMode int

	// Amount from the triggering event (e.g. damage dealt) for triggered abilities
	currentEventAmount int

	// SourceID of the triggering event
	currentEventSourceID uuid.UUID

	// Card currently being resolved from the stack
	resolvingCard Card

	// Color state installed before target legality check during resolution
	resolvingColorSourceID uuid.UUID
	resolvingColorOverride *[]Color

	// Zone the resolving spell was cast from
	resolvingCastZone Zone

	// Cast-time snapshot for the resolving stack object (CR 608.2g)
	resolvingCastContext *CastContext

	// Targets of the spell currently being resolved
	resolvingTargets []uuid.UUID

	// Divided damage distribution chosen at cast/activation
	resolvingDamageDistribution map[uuid.UUID]int

	// Counter distribution assigned at cast/activation
	resolvingCounterDistribution map[uuid.UUID]int

	// ID of the most recently sacrificed permanent paid as a cost
	lastSacrificedID uuid.UUID

	// Most recent card revealed by a RevealFromHandCost
	lastCostReveal Card

	// Most recently exiled card recorded during cost payment or effect resolution
	lastExiledCard Card
}

// NewResolutionState creates an empty ResolutionState.
func NewResolutionState() ResolutionState {
	return ResolutionState{
		resolvingCastZone: ZoneAny,
	}
}

// Begin installs the stack object's resolution context and returns a deferred cleanup func.
func (s *ResolutionState) Begin(obj *StackObject) func() {
	if obj == nil {
		return func() {}
	}
	s.currentX = obj.XValue
	s.currentMode = obj.ModeChoice
	s.currentEventAmount = obj.EventAmount
	s.currentEventSourceID = obj.EventSourceID
	s.resolvingCard = obj.Card
	s.resolvingTargets = obj.Targets
	s.resolvingDamageDistribution = obj.DamageDistribution
	s.resolvingCounterDistribution = obj.CounterDistribution
	s.resolvingCastZone = obj.CastZone
	s.resolvingCastContext = obj.CastContext

	return func() {
		s.Clear()
	}
}

// SetColorOverride installs a temporary color override and returns a cleanup func to restore the previous values.
func (s *ResolutionState) SetColorOverride(sourceID uuid.UUID, override *[]Color) func() {
	prevID := s.resolvingColorSourceID
	prevOverride := s.resolvingColorOverride
	s.resolvingColorSourceID = sourceID
	s.resolvingColorOverride = override
	return func() {
		s.resolvingColorSourceID = prevID
		s.resolvingColorOverride = prevOverride
	}
}

// Clear resets all resolving state fields to zero values.
func (s *ResolutionState) Clear() {
	s.currentX = 0
	s.currentMode = 0
	s.currentEventAmount = 0
	s.currentEventSourceID = uuid.Nil
	s.resolvingCard = nil
	s.resolvingTargets = nil
	s.resolvingDamageDistribution = nil
	s.resolvingCounterDistribution = nil
	s.resolvingCastZone = ZoneAny
	s.resolvingCastContext = nil
}

// ClearDistributions clears the damage and counter distributions after effects have run.
func (s *ResolutionState) ClearDistributions() {
	s.resolvingDamageDistribution = nil
	s.resolvingCounterDistribution = nil
}

// Getters and Setters

func (s *ResolutionState) X() int                                 { return s.currentX }
func (s *ResolutionState) SetX(x int)                             { s.currentX = x }
func (s *ResolutionState) Mode() int                              { return s.currentMode }
func (s *ResolutionState) SetMode(m int)                          { s.currentMode = m }
func (s *ResolutionState) EventAmount() int                       { return s.currentEventAmount }
func (s *ResolutionState) SetEventAmount(a int)                   { s.currentEventAmount = a }
func (s *ResolutionState) EventSourceID() uuid.UUID               { return s.currentEventSourceID }
func (s *ResolutionState) SetEventSourceID(id uuid.UUID)          { s.currentEventSourceID = id }
func (s *ResolutionState) ResolvingCard() Card                    { return s.resolvingCard }
func (s *ResolutionState) SetResolvingCard(c Card)                { s.resolvingCard = c }
func (s *ResolutionState) ColorSourceID() uuid.UUID               { return s.resolvingColorSourceID }
func (s *ResolutionState) ColorOverride() *[]Color                { return s.resolvingColorOverride }
func (s *ResolutionState) ResolvingCastZone() Zone                { return s.resolvingCastZone }
func (s *ResolutionState) SetResolvingCastZone(z Zone)            { s.resolvingCastZone = z }
func (s *ResolutionState) ResolvingCastContext() *CastContext     { return s.resolvingCastContext }
func (s *ResolutionState) SetResolvingCastContext(c *CastContext) { s.resolvingCastContext = c }
func (s *ResolutionState) ResolvingTargets() []uuid.UUID          { return s.resolvingTargets }
func (s *ResolutionState) SetResolvingTargets(t []uuid.UUID)      { s.resolvingTargets = t }
func (s *ResolutionState) DamageDistribution() map[uuid.UUID]int {
	return s.resolvingDamageDistribution
}
func (s *ResolutionState) SetDamageDistribution(d map[uuid.UUID]int) {
	s.resolvingDamageDistribution = d
}
func (s *ResolutionState) CounterDistribution() map[uuid.UUID]int {
	return s.resolvingCounterDistribution
}
func (s *ResolutionState) SetCounterDistribution(d map[uuid.UUID]int) {
	s.resolvingCounterDistribution = d
}
func (s *ResolutionState) LastSacrificedID() uuid.UUID      { return s.lastSacrificedID }
func (s *ResolutionState) SetLastSacrificedID(id uuid.UUID) { s.lastSacrificedID = id }
func (s *ResolutionState) LastCostReveal() Card             { return s.lastCostReveal }
func (s *ResolutionState) SetLastCostReveal(c Card)         { s.lastCostReveal = c }
func (s *ResolutionState) LastExiledCard() Card             { return s.lastExiledCard }
func (s *ResolutionState) SetLastExiledCard(c Card)         { s.lastExiledCard = c }

// Clone creates an independent deep copy of ResolutionState.
func (s *ResolutionState) Clone() ResolutionState {
	c := ResolutionState{
		currentX:               s.currentX,
		currentMode:            s.currentMode,
		currentEventAmount:     s.currentEventAmount,
		currentEventSourceID:   s.currentEventSourceID,
		resolvingCard:          s.resolvingCard,
		resolvingColorSourceID: s.resolvingColorSourceID,
		resolvingCastZone:      s.resolvingCastZone,
		resolvingCastContext:   s.resolvingCastContext,
		lastSacrificedID:       s.lastSacrificedID,
		lastCostReveal:         s.lastCostReveal,
		lastExiledCard:         s.lastExiledCard,
	}
	if s.resolvingColorOverride != nil {
		colors := append([]Color(nil), (*s.resolvingColorOverride)...)
		c.resolvingColorOverride = &colors
	}
	if len(s.resolvingTargets) > 0 {
		c.resolvingTargets = append([]uuid.UUID(nil), s.resolvingTargets...)
	}
	if len(s.resolvingDamageDistribution) > 0 {
		c.resolvingDamageDistribution = make(map[uuid.UUID]int, len(s.resolvingDamageDistribution))
		maps.Copy(c.resolvingDamageDistribution, s.resolvingDamageDistribution)
	}
	if len(s.resolvingCounterDistribution) > 0 {
		c.resolvingCounterDistribution = make(map[uuid.UUID]int, len(s.resolvingCounterDistribution))
		maps.Copy(c.resolvingCounterDistribution, s.resolvingCounterDistribution)
	}
	return c
}
