package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ResolutionState encapsulates transient context for spells and abilities
// that are currently resolving from the stack or being processed during costs.
type ResolutionState struct {
	manaProduced          int
	manaColors            []Color
	manaTracking          bool
	targetSpecs           []Target
	targetIncarnations    map[uuid.UUID]uuid.UUID
	permanentBindings     map[*PermanentBinding]capturedPermanent
	createdTokenBindings  map[*CreatedTokensBinding][]capturedPermanent
	cardBindings          map[*CardBinding]capturedCard
	cardsBindings         map[*CardsBinding][]capturedCard
	spellBindings         map[*SpellBinding]uuid.UUID
	playerBindings        map[*PlayerBinding]uuid.UUID
	numberBindings        map[*NumberBinding]int
	manaTypesBindings     map[*ManaTypesBinding][]Color
	damageSourceBindings  map[*DamageSourceBinding]DamageSource
	damageResultsBindings map[*DamageResultsBinding][]damageOutcome
	damageResultBindings  map[*DamageResultBinding]damageOutcome
	sourceIncarnation     uuid.UUID
	sourceCaptured        bool
	// X value for the currently resolving spell or activation
	currentX int

	// Chosen mode for the currently resolving modal spell (0-indexed)
	currentMode int

	// Amount from the triggering event (e.g. damage dealt) for triggered abilities
	currentEventAmount int

	// SourceID of the triggering event
	currentEventSourceID uuid.UUID

	// Card currently being resolved from the stack
	resolvingCard       Card
	resolvingController uuid.UUID

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
	s.targetSpecs = obj.TargetSpecs
	s.targetIncarnations = obj.targetIncarnations
	s.currentX = obj.XValue
	s.currentMode = obj.ModeChoice
	s.currentEventAmount = obj.EventAmount
	s.currentEventSourceID = obj.EventSourceID
	s.resolvingCard = obj.Card
	s.resolvingController = obj.Controller
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
	s.targetSpecs = nil
	s.targetIncarnations = nil
	s.permanentBindings = nil
	s.createdTokenBindings = nil
	s.cardBindings = nil
	s.cardsBindings = nil
	s.spellBindings = nil
	s.playerBindings = nil
	s.numberBindings = nil
	s.manaTypesBindings = nil
	s.damageSourceBindings = nil
	s.damageResultsBindings = nil
	s.damageResultBindings = nil
	s.currentX = 0
	s.currentMode = 0
	s.currentEventAmount = 0
	s.currentEventSourceID = uuid.Nil
	s.resolvingCard = nil
	s.resolvingController = uuid.Nil
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
		manaProduced:           s.manaProduced,
		manaColors:             append([]Color(nil), s.manaColors...),
		manaTracking:           s.manaTracking,
		permanentBindings:      maps.Clone(s.permanentBindings),
		createdTokenBindings:   cloneCreatedTokenBindings(s.createdTokenBindings),
		cardBindings:           maps.Clone(s.cardBindings),
		cardsBindings:          cloneCardsBindings(s.cardsBindings),
		spellBindings:          maps.Clone(s.spellBindings),
		playerBindings:         maps.Clone(s.playerBindings),
		numberBindings:         maps.Clone(s.numberBindings),
		manaTypesBindings:      cloneManaTypesBindings(s.manaTypesBindings),
		targetSpecs:            append([]Target(nil), s.targetSpecs...),
		targetIncarnations:     maps.Clone(s.targetIncarnations),
		sourceIncarnation:      s.sourceIncarnation,
		sourceCaptured:         s.sourceCaptured,
		currentX:               s.currentX,
		currentMode:            s.currentMode,
		currentEventAmount:     s.currentEventAmount,
		currentEventSourceID:   s.currentEventSourceID,
		resolvingCard:          s.resolvingCard,
		resolvingController:    s.resolvingController,
		resolvingColorSourceID: s.resolvingColorSourceID,
		resolvingCastZone:      s.resolvingCastZone,
		resolvingCastContext:   s.resolvingCastContext,
		lastSacrificedID:       s.lastSacrificedID,
		lastCostReveal:         s.lastCostReveal,
		lastExiledCard:         s.lastExiledCard,
	}
	if len(s.damageSourceBindings) > 0 {
		c.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource, len(s.damageSourceBindings))
		for binding, source := range s.damageSourceBindings {
			c.damageSourceBindings[binding] = cloneDamageSource(source)
		}
	}
	if len(s.damageResultsBindings) > 0 {
		c.damageResultsBindings = make(map[*DamageResultsBinding][]damageOutcome, len(s.damageResultsBindings))
		for binding, outcomes := range s.damageResultsBindings {
			c.damageResultsBindings[binding] = cloneDamageOutcomes(outcomes)
		}
	}
	if len(s.damageResultBindings) > 0 {
		c.damageResultBindings = make(map[*DamageResultBinding]damageOutcome, len(s.damageResultBindings))
		for binding, outcome := range s.damageResultBindings {
			c.damageResultBindings[binding] = cloneDamageOutcome(outcome)
		}
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

func cloneCardsBindings(bindings map[*CardsBinding][]capturedCard) map[*CardsBinding][]capturedCard {
	if bindings == nil {
		return nil
	}
	clone := make(map[*CardsBinding][]capturedCard, len(bindings))
	for binding, cards := range bindings {
		clone[binding] = append([]capturedCard(nil), cards...)
	}
	return clone
}
