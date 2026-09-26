package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type receivesDamagePattern struct {
	amount            *NumberBinding
	controller        *PlayerBinding
	captureController bool
}

// ReceivesDamage saves positive damage dealt to the source permanent.
func ReceivesDamage(_ sourceRef, amount *NumberBinding) receivesDamagePattern {
	return receivesDamagePattern{amount: amount}
}

// WithSourceController saves the damage source controller at event time.
func (p receivesDamagePattern) WithSourceController(controller *PlayerBinding) receivesDamagePattern {
	p.controller, p.captureController = controller, true
	return p
}

func (receivesDamagePattern) eventPattern() {}
func (receivesDamagePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtDamageDealt, false, effect).SetConditionData(EventTargetIsSelf{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p receivesDamagePattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source receives damage", battlefieldSource: true,
		children: []definitionEdge{{"amount", &eventAmountCapture{binding: p.amount}}}}
	if p.captureController {
		n.children = append(n.children, definitionEdge{"source controller", &eventPlayerCapture{binding: p.controller}})
	}
	return n
}
func (p receivesDamagePattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []ResolvingEffect{&eventAmountCapture{binding: p.amount, amount: event.Amount}}
	if p.captureController {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: event.SourceControllerID})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
