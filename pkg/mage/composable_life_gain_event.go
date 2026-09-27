package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type controllerLifeGainEventPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	captureAmount   bool
	duplicateAmount bool
}

// ControllerGainsLife matches positive life gain by the source controller.
// It saves the player at detection without declaring a target.
func ControllerGainsLife(player *PlayerBinding) controllerLifeGainEventPattern {
	return controllerLifeGainEventPattern{player: player}
}

// WithAmount saves the amount of life gained when the event occurs.
func (p controllerLifeGainEventPattern) WithAmount(amount *NumberBinding) controllerLifeGainEventPattern {
	if p.captureAmount {
		p.duplicateAmount = true
	}
	p.amount, p.captureAmount = amount, true
	return p
}

func (controllerLifeGainEventPattern) eventPattern() {}
func (controllerLifeGainEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtLifeGained, false, effect).SetConditionData(EventPlayerIsController{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p controllerLifeGainEventPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "you gain life", battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.captureAmount {
		n.bindingWrites = append(n.bindingWrites, p.amount)
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
		if p.duplicateAmount {
			n.problem = "life gain event: supply one amount binding"
		}
	}
	return n
}
func (p controllerLifeGainEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []Effect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	return []Effect{Sequence(append(captures, Sequence(effects...))...)}
}
