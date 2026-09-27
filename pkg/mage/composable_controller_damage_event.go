package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type controllerDamageEventPattern struct {
	amount          *NumberBinding
	source          *DamageSourceBinding
	captureSource   bool
	duplicateSource bool
}

// ControllerIsDealtDamage saves damage dealt to the source controller.
func ControllerIsDealtDamage(amount *NumberBinding) controllerDamageEventPattern {
	return controllerDamageEventPattern{amount: amount}
}

// WithSource saves the damage source when damage occurs.
func (p controllerDamageEventPattern) WithSource(source *DamageSourceBinding) controllerDamageEventPattern {
	if p.captureSource {
		p.duplicateSource = true
	}
	p.source, p.captureSource = source, true
	return p
}

func (controllerDamageEventPattern) eventPattern() {}
func (controllerDamageEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtDamageDealt, false, effect).SetConditionData(EventTargetIsController{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p controllerDamageEventPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "you are dealt damage", battlefieldSource: true,
		children: []definitionEdge{{"amount", &eventAmountCapture{binding: p.amount}}}}
	if p.captureSource {
		if p.source == nil || p.duplicateSource {
			n.problem = "controller damage event requires one source binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.source)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.source}})
		}
	}
	return n
}
func (p controllerDamageEventPattern) captureDetectedEvent(_ *core.GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureSource || p.source == nil {
		return nil
	}
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	return []Effect{&eventDamageSourceCapture{binding: p.source, source: cloneDamageSource(game.damageEventSource)}}
}
func (p controllerDamageEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventAmountCapture{binding: p.amount, amount: event.Amount}, Sequence(effects...))}
}
