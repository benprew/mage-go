package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type controllerDrawEventPattern struct{ player *PlayerBinding }

// ControllerDrawsCard matches a card drawn by the source controller.
// It saves the player at detection without declaring a target.
func ControllerDrawsCard(player *PlayerBinding) EventPattern {
	return controllerDrawEventPattern{player: player}
}

func (controllerDrawEventPattern) eventPattern() {}
func (controllerDrawEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtCardDrawn, false, effect).SetConditionData(EventPlayerIsController{})
}
func (p controllerDrawEventPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "you draw a card", battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
}
func (p controllerDrawEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.player, event.PlayerID}, Sequence(effects...))}
}
