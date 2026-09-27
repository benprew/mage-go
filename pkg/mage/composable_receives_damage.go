package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type receivesDamagePattern struct {
	amount            *NumberBinding
	controller        *PlayerBinding
	captureController bool
	source            *DamageSourceBinding
	captureSource     bool
	duplicateSource   bool
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

// WithSource saves the damage source when damage occurs.
func (p receivesDamagePattern) WithSource(source *DamageSourceBinding) receivesDamagePattern {
	if p.captureSource {
		p.duplicateSource = true
	}
	p.source, p.captureSource = source, true
	return p
}

func (receivesDamagePattern) eventPattern() {}
func (receivesDamagePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtDamageDealt, false, effect).SetConditionData(EventTargetIsSelf{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p receivesDamagePattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source receives damage", battlefieldSource: true,
		bindingWrite: p.amount, children: []definitionEdge{{"amount", &eventAmountCapture{binding: p.amount}}}}
	if p.captureController {
		n.bindingWrites = append(n.bindingWrites, p.controller)
		n.children = append(n.children, definitionEdge{"source controller", &eventPlayerCapture{binding: p.controller}})
	}
	if p.captureSource {
		if p.source == nil || p.duplicateSource {
			n.problem = "damage event requires one source binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.source)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.source}})
		}
	}
	return n
}
func (p receivesDamagePattern) captureDetectedEvent(_ *core.GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureSource || p.source == nil {
		return nil
	}
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	return []Effect{&eventDamageSourceCapture{binding: p.source, source: cloneDamageSource(game.damageEventSource)}}
}
func (p receivesDamagePattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []ResolvingEffect{&eventAmountCapture{binding: p.amount, amount: event.Amount}}
	if p.captureController {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: event.SourceControllerID})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
