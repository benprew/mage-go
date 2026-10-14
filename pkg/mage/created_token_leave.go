package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CreatedTokenLeaves matches a linked token's departure from the battlefield.
// The source's current visit supplies the group, including simultaneous moves.
func CreatedTokenLeaves(link *CreatedTokenLink) EventPattern {
	return createdTokenLeavePattern{link: link}
}

type createdTokenLeavePattern struct{ link *CreatedTokenLink }

func (createdTokenLeavePattern) eventPattern() {}
func (p createdTokenLeavePattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "a linked token leaves the battlefield", battlefieldSource: true, children: []definitionEdge{{"link", p.link}}}
}
func (p createdTokenLeavePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtZoneChange, false, effect).AndConditionData(p)
}
func (p createdTokenLeavePattern) CheckTriggerCond(event *core.GameEvent, reader GameReader, sourceID, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	source := game.FindPermanent(sourceID)
	if source == nil || source.PhasedOut {
		return false
	}
	leaving := departureEventPermanent(event, game)
	return leaving != nil && leaving.IsToken && leaving.createdTokenLink == p.link && leaving.createdTokenSource == (capturedPermanent{source.ID(), source.incarnationID})
}

// departureEventPermanent reads the event's exact visit from the former state.
// A returned object must not replace the departure snapshot.
func departureEventPermanent(event *core.GameEvent, game *Game) *Permanent {
	if event.Type != core.EvtZoneChange || event.FromZone != core.ZoneBattlefield || event.ToZone == core.ZoneBattlefield || event.SourceIncarnation == uuid.Nil {
		return nil
	}
	permanent, lki := game.readPermanentIncarnationInformation(capturedPermanent{event.SourceID, event.SourceIncarnation})
	if lki != nil {
		permanent = lki.Snapshot
	}
	if permanent == nil || permanent.PhasedOut {
		return nil
	}
	return permanent
}
