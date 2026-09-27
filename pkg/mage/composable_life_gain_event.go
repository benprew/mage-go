package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type lifeGainEventPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	captureAmount   bool
	duplicateAmount bool
	controllerOnly  bool
}

// ControllerGainsLife matches positive life gain by the source controller.
// It saves the player at detection without declaring a target.
func ControllerGainsLife(player *PlayerBinding) lifeGainEventPattern {
	return lifeGainEventPattern{player: player, controllerOnly: true}
}

// PlayerGainsLife saves any player who gains a positive amount of life.
func PlayerGainsLife(player *PlayerBinding) lifeGainEventPattern {
	return lifeGainEventPattern{player: player}
}

// WithAmount saves the amount of life gained when the event occurs.
func (p lifeGainEventPattern) WithAmount(amount *NumberBinding) lifeGainEventPattern {
	if p.captureAmount {
		p.duplicateAmount = true
	}
	p.amount, p.captureAmount = amount, true
	return p
}

func (lifeGainEventPattern) eventPattern() {}
func (p lifeGainEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	trigger := NewTriggered(core.EvtLifeGained, false, effect).SetConditionData(EventAmountGreaterThan{N: 0})
	if p.controllerOnly {
		trigger.AndConditionData(EventPlayerIsController{})
	}
	return trigger
}
func (p lifeGainEventPattern) inspectDefinition() definitionNode {
	text := "a player gains life"
	if p.controllerOnly {
		text = "you gain life"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
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
func (p lifeGainEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []Effect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	return []Effect{Sequence(append(captures, Sequence(effects...))...)}
}
