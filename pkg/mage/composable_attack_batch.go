package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type controlledAttackBatchPattern struct {
	controlled     bool
	count          *NumberBinding
	captureCount   bool
	duplicateCount bool
}

// OneOrMoreCreaturesYouControlAttack matches one legal declaration group.
func OneOrMoreCreaturesYouControlAttack() controlledAttackBatchPattern {
	return controlledAttackBatchPattern{controlled: true}
}

// OneOrMoreCreaturesAttack matches one attack declaration by either player.
func OneOrMoreCreaturesAttack() controlledAttackBatchPattern {
	return controlledAttackBatchPattern{}
}

// WithCount saves the number of creatures declared in this attack group.
func (p controlledAttackBatchPattern) WithCount(count *NumberBinding) controlledAttackBatchPattern {
	if p.captureCount {
		p.duplicateCount = true
	}
	p.count = count
	p.captureCount = true
	return p
}

func (controlledAttackBatchPattern) eventPattern() {}

func (p controlledAttackBatchPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.controlled {
		return NewTriggered(EvtAttackersDeclared, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventPlayerIsController{}, EventAmountGreaterThan{N: 0},
		}})
	}
	return NewTriggered(EvtAttackersDeclared, false, effect).SetConditionData(EventAmountGreaterThan{N: 0})
}

func (p controlledAttackBatchPattern) inspectDefinition() definitionNode {
	text := "one or more creatures attack"
	if p.controlled {
		text = "one or more creatures you control attack"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true}
	if p.captureCount {
		if p.count == nil || p.duplicateCount {
			n.problem = "attack group: supply one count binding"
		} else {
			n.bindingWrite = p.count
			n.children = append(n.children, definitionEdge{"count", &eventAmountCapture{binding: p.count}})
		}
	}
	return n
}

func (p controlledAttackBatchPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureCount {
		return effects
	}
	return []Effect{Sequence(&eventAmountCapture{binding: p.count, amount: event.Amount}, Sequence(effects...))}
}
