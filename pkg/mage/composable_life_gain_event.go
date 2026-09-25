package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type controllerLifeGainEventPattern struct{ player *PlayerBinding }

// ControllerGainsLife matches positive life gain by the source controller.
// It saves the player at detection without declaring a target.
func ControllerGainsLife(player *PlayerBinding) EventPattern {
	return controllerLifeGainEventPattern{player: player}
}

func (controllerLifeGainEventPattern) eventPattern() {}
func (controllerLifeGainEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtLifeGained, false, effect).SetConditionData(EventPlayerIsController{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p controllerLifeGainEventPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "you gain life", battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
}
func (p controllerLifeGainEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.player, event.PlayerID}, Sequence(effects...))}
}
