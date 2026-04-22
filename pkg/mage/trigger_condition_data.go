package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// TriggerConditionData is a composable, data-driven predicate for trigger
// conditions. Each implementation checks one aspect of the event/game state.
// Compose with AndTriggerCond, OrTriggerCond, NotTriggerCond.
//
// Use AsTriggerCondition() to bridge into the closure-based TriggerCondition type.
type TriggerConditionData interface {
	CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
}

// AsTriggerCondition wraps a TriggerConditionData into a TriggerCondition closure.
func AsTriggerCondition(d TriggerConditionData) TriggerCondition {
	return func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
		return d.CheckTriggerCond(evt, g, sourceID, controllerID)
	}
}

// ---------------------------------------------------------------------------
// Atomic predicates
// ---------------------------------------------------------------------------

// EventSourceIsSelf checks evt.SourceID == sourceID.
type EventSourceIsSelf struct{}

func (EventSourceIsSelf) CheckTriggerCond(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
	return evt.SourceID == sourceID
}

// EventTargetIsSelf checks evt.TargetID == sourceID.
type EventTargetIsSelf struct{}

func (EventTargetIsSelf) CheckTriggerCond(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
	return evt.TargetID == sourceID
}

// EventPlayerIsController checks evt.PlayerID == controllerID.
type EventPlayerIsController struct{}

func (EventPlayerIsController) CheckTriggerCond(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
	return evt.PlayerID == controllerID
}

// EventPlayerIsNotController checks evt.PlayerID != controllerID.
type EventPlayerIsNotController struct{}

func (EventPlayerIsNotController) CheckTriggerCond(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
	return evt.PlayerID != controllerID
}

// EventSourceNotSelf checks evt.SourceID != sourceID ("another" creature).
type EventSourceNotSelf struct{}

func (EventSourceNotSelf) CheckTriggerCond(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
	return evt.SourceID != sourceID
}

// EventSourceControlledByController checks that the permanent referenced by
// evt.SourceID is controlled by the trigger's controller.
type EventSourceControlledByController struct{}

func (EventSourceControlledByController) CheckTriggerCond(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	perm := g.FindPermanent(evt.SourceID)
	return perm != nil && perm.Controller == controllerID
}

// EventSourceControlledByOpponent checks that the permanent referenced by
// evt.SourceID is NOT controlled by the trigger's controller.
type EventSourceControlledByOpponent struct{}

func (EventSourceControlledByOpponent) CheckTriggerCond(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	perm := g.FindPermanent(evt.SourceID)
	return perm != nil && perm.Controller != controllerID
}

// EventSourceHasType checks that the permanent at evt.SourceID has a card type.
type EventSourceHasType struct {
	Type CardType
}

func (c EventSourceHasType) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	perm := g.FindPermanent(evt.SourceID)
	return perm != nil && perm.HasType(c.Type)
}

// EventSourceHasSubType checks that the permanent at evt.SourceID has a subtype.
type EventSourceHasSubType struct {
	SubType string
}

func (c EventSourceHasSubType) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	perm := g.FindPermanent(evt.SourceID)
	return perm != nil && perm.HasSubType(c.SubType)
}

// EventSourceMatchesPermanentFilter checks the permanent at evt.SourceID against a filter.
type EventSourceMatchesPermanentFilter struct {
	Filter PermanentFilter
}

func (c EventSourceMatchesPermanentFilter) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	perm := g.FindPermanent(evt.SourceID)
	return perm != nil && c.Filter.Match(perm, g.(*Game))
}

// SourceIsAttachedToEventSource checks that the source permanent is attached
// to the permanent in evt.SourceID (for aura triggers like "when enchanted
// creature becomes tapped").
type SourceIsAttachedToEventSource struct{}

func (SourceIsAttachedToEventSource) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return false
	}
	return src.AttachedTo == evt.SourceID
}

// SourceIsTapped checks that the source permanent is tapped.
type SourceIsTapped struct{}

func (SourceIsTapped) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && perm.Tapped
}

// SourceIsUntapped checks that the source permanent is untapped.
type SourceIsUntapped struct{}

func (SourceIsUntapped) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && !perm.Tapped
}

// SourceNotSummonSick checks the source doesn't have summoning sickness.
type SourceNotSummonSick struct{}

func (SourceNotSummonSick) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && !perm.HasAttr(AttrSummonSick)
}

// EventSourceDamagedBySource checks if the creature that died (evt.SourceID)
// was damaged by sourceID this turn (Sengir Vampire pattern).
type EventSourceDamagedBySource struct{}

func (EventSourceDamagedBySource) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	sources := g.GetDamageSources(evt.SourceID)
	return sources[sourceID]
}

// EventIsChosenPlayerUpkeep checks evt.PlayerID matches the source permanent's ChosenPlayer.
type EventIsChosenPlayerUpkeep struct{}

func (EventIsChosenPlayerUpkeep) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && evt.PlayerID == perm.ChosenPlayer
}

// HasAttackedThisTurnCond checks if the source has attacked this turn.
type HasAttackedThisTurnCond struct{}

func (HasAttackedThisTurnCond) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return g.HasAttackedThisTurn(sourceID)
}

// EventFlagIsFalse checks that evt.Flag is false (used to distinguish tap-cost
// abilities from non-tap abilities).
type EventFlagIsFalse struct{}

func (EventFlagIsFalse) CheckTriggerCond(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return !evt.Flag
}

// ---------------------------------------------------------------------------
// Combinators
// ---------------------------------------------------------------------------

// AndTriggerCond requires all inner conditions to be true.
type AndTriggerCond struct {
	Conditions []TriggerConditionData
}

func (c AndTriggerCond) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	for _, inner := range c.Conditions {
		if !inner.CheckTriggerCond(evt, g, sourceID, controllerID) {
			return false
		}
	}
	return true
}

// OrTriggerCond requires any inner condition to be true.
type OrTriggerCond struct {
	Conditions []TriggerConditionData
}

func (c OrTriggerCond) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	for _, inner := range c.Conditions {
		if inner.CheckTriggerCond(evt, g, sourceID, controllerID) {
			return true
		}
	}
	return false
}

// NotTriggerCond negates an inner condition.
type NotTriggerCond struct {
	Inner TriggerConditionData
}

func (c NotTriggerCond) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	return !c.Inner.CheckTriggerCond(evt, g, sourceID, controllerID)
}
