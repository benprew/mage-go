package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type landPlayPattern struct {
	player     *PlayerBinding
	land       *PermanentBinding
	ordinal    *NumberBinding
	afterFirst bool
}

// ControllerPlaysLand saves the player, land visit and turn ordinal of a land play.
func ControllerPlaysLand(player *PlayerBinding, land *PermanentBinding, ordinal *NumberBinding) landPlayPattern {
	return landPlayPattern{player: player, land: land, ordinal: ordinal}
}

// AfterFirst excludes the player's first land play in that turn.
// The event ordinal is a historical fact. Later state cannot change it.
func (p landPlayPattern) AfterFirst() landPlayPattern { p.afterFirst = true; return p }
func (landPlayPattern) eventPattern()                 {}
func (p landPlayPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtLandPlayed, false, effect).SetConditionData(controllerLandPlay{afterFirst: p.afterFirst})
}
func (p landPlayPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "you play a land", battlefieldSource: true,
		bindingWrites: []savedBinding{p.player, p.land, p.ordinal},
		children: []definitionEdge{
			{"player", &eventPlayerCapture{binding: p.player}},
			{"land", &eventPermanentCapture{binding: p.land}},
			{"ordinal", &eventAmountCapture{binding: p.ordinal}},
		}}
	if p.player == nil || p.land == nil || p.ordinal == nil {
		n.problem = "land play requires a player, permanent and ordinal binding"
	}
	return n
}
func (p landPlayPattern) captureDetectedEvent(event *core.GameEvent, _ GameReader, _ uuid.UUID) []Effect {
	return []Effect{
		&eventPlayerCapture{binding: p.player, player: event.PlayerID},
		&eventPermanentCapture{binding: p.land, value: capturedPermanent{event.SourceID, event.SourceIncarnation}},
		&eventAmountCapture{binding: p.ordinal, amount: event.Amount},
	}
}

type controllerLandPlay struct{ afterFirst bool }

func (c controllerLandPlay) CheckTriggerCond(event *core.GameEvent, g GameReader, _, controller uuid.UUID) bool {
	return event.Type == core.EvtLandPlayed && event.PlayerID == controller && g.GetPlayer(event.PlayerID) != nil && event.Amount > 0 && (!c.afterFirst || event.Amount > 1)
}
