package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type selfDiscardEffectPattern struct{ cause *PlayerBinding }

// ThisCardDiscardedByOpponentEffect saves the controller of the discard effect.
// Hidden destinations cannot supply the card's abilities.
func ThisCardDiscardedByOpponentEffect(cause *PlayerBinding) EventPattern {
	return selfDiscardEffectPattern{cause: cause}
}

func (selfDiscardEffectPattern) eventPattern() {}
func (p selfDiscardEffectPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "an opponent's effect discards this card", bindingWrite: p.cause, children: []definitionEdge{{"cause-controller", p.cause}}}
}
func (p selfDiscardEffectPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	trigger := NewTriggered(core.EvtDiscard, false, effect)
	trigger.sourceZone = core.ZoneGraveyard
	trigger.InZone(core.ZoneExile)
	trigger.SetCondition(func(event *core.GameEvent, _ GameReader, source, controller uuid.UUID) bool {
		return event.SourceID == source && event.PlayerID == controller && event.DiscardByEffect && event.DiscardedHandVisit != uuid.Nil && event.DiscardCauseID != uuid.Nil && event.DiscardCauseController != uuid.Nil && event.DiscardCauseController != controller
	})
	return trigger
}
func (p selfDiscardEffectPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.cause, event.DiscardCauseController}, Sequence(effects...))}
}

func (g *Game) snapshotDiscardCause(action *DiscardAction) {
	if action.handVisit == uuid.Nil {
		action.handVisit = g.handVisit(action.CardID())
	}
	action.discardEvent = core.GameEvent{Type: core.EvtDiscard, SourceID: action.CardID(), PlayerID: action.PlayerID(), DiscardByEffect: action.CausedByEffect(), DiscardedHandVisit: action.handVisit}
	if !action.CausedByEffect() || action.ActionSource() == uuid.Nil {
		return
	}
	action.discardEvent.DiscardCauseID = action.ActionSource()
	for _, object := range slices.Backward(g.stack.resolvingReferenceObjects) {
		if object.SourceID == action.ActionSource() {
			action.discardEvent.DiscardCauseID = object.ID
			action.discardEvent.DiscardCauseController = object.Controller
			return
		}
	}
	if source := g.FindPermanent(action.ActionSource()); source != nil && !source.PhasedOut {
		action.discardEvent.DiscardCauseController = source.ControllerID()
	}
}

func (g *Game) completedDiscardEvent(action *DiscardAction) core.GameEvent {
	event := action.discardEvent
	event.DiscardedCardVisit = g.graveyardVisit(action.CardID())
	return event
}
