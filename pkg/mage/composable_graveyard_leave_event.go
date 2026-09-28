package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type graveyardLeaveBatchPattern struct {
	count          *NumberBinding
	captureCount   bool
	duplicateCount bool
}

// OneOrMoreCardsLeaveYourGraveyard matches one graveyard leave event group.
func OneOrMoreCardsLeaveYourGraveyard() graveyardLeaveBatchPattern {
	return graveyardLeaveBatchPattern{}
}

// WithCount saves the number of cards that left in the event group.
func (p graveyardLeaveBatchPattern) WithCount(count *NumberBinding) graveyardLeaveBatchPattern {
	if p.captureCount {
		p.duplicateCount = true
	}
	p.count = count
	p.captureCount = true
	return p
}

func (graveyardLeaveBatchPattern) eventPattern() {}

func (graveyardLeaveBatchPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtCardsLeftGraveyard, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventPlayerIsController{}, EventAmountGreaterThan{N: 0},
	}})
}

func (p graveyardLeaveBatchPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "one or more cards leave your graveyard", battlefieldSource: true}
	if p.captureCount {
		if p.count == nil || p.duplicateCount {
			n.problem = "graveyard leave event: supply one count binding"
		} else {
			n.bindingWrite = p.count
			n.children = append(n.children, definitionEdge{"count", &eventAmountCapture{binding: p.count}})
		}
	}
	return n
}

func (p graveyardLeaveBatchPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	if !p.captureCount {
		return effects
	}
	return []Effect{Sequence(&eventAmountCapture{binding: p.count, amount: event.Amount}, Sequence(effects...))}
}
