package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type controllerLastCounterPattern struct {
	counter core.CounterType
	actor   *PlayerBinding
}

// ControllerRemovesLastCounter matches actual removal from the source visit.
// It saves the acting player before later control or zone changes.
func ControllerRemovesLastCounter(counter core.CounterType, actor *PlayerBinding) EventPattern {
	return controllerLastCounterPattern{counter: counter, actor: actor}
}

func (controllerLastCounterPattern) eventPattern() {}

func (p controllerLastCounterPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtCountersRemoved, false, effect).SetConditionData(sourceLastCounterRemoval{counter: p.counter})
}

func (p controllerLastCounterPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "you remove the last counter from this permanent", battlefieldSource: true,
		bindingWrite: p.actor, children: []definitionEdge{{"actor", p.actor}}}
	if p.actor == nil {
		n.problem = "supply a non-nil acting player binding"
	} else if p.counter < 0 || p.counter >= core.NumCounters {
		n.problem = "supply a valid counter type"
	}
	return n
}

func (p controllerLastCounterPattern) captureDetectedEvent(event *core.GameEvent, _ GameReader, _ uuid.UUID) []Effect {
	return []Effect{&eventPlayerCapture{binding: p.actor, player: event.PlayerID}}
}

type sourceLastCounterRemoval struct{ counter core.CounterType }

func (c sourceLastCounterRemoval) CheckTriggerCond(event *core.GameEvent, g GameReader, source, controller uuid.UUID) bool {
	if event.Type != core.EvtCountersRemoved || event.TargetID != source || event.CounterType != c.counter || event.Amount <= 0 || event.CountersRemaining != 0 || event.PlayerID == uuid.Nil || event.PlayerID != controller {
		return false
	}
	p := g.FindPermanent(source)
	return p != nil && !p.PhasedOut && event.TargetIncarnation == p.incarnationID
}
