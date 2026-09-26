package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type controllerDamageEventPattern struct{ amount *NumberBinding }

// ControllerIsDealtDamage saves damage dealt to the source controller.
func ControllerIsDealtDamage(amount *NumberBinding) EventPattern {
	return controllerDamageEventPattern{amount: amount}
}

func (controllerDamageEventPattern) eventPattern() {}
func (controllerDamageEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtDamageDealt, false, effect).SetConditionData(EventTargetIsController{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p controllerDamageEventPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "you are dealt damage", battlefieldSource: true,
		children: []definitionEdge{{"amount", &eventAmountCapture{binding: p.amount}}}}
}
func (p controllerDamageEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventAmountCapture{binding: p.amount, amount: event.Amount}, Sequence(effects...))}
}
