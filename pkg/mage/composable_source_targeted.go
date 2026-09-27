package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type sourceTargetedBySpellPattern struct{}
type sourceTargetedPattern struct {
	opponentOnly        bool
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
}

// BecomesTargeted matches a spell or ability that targets the battlefield source.
func BecomesTargeted(sourceRef) sourceTargetedPattern { return sourceTargetedPattern{} }

// ByOpponent checks the targeting object's controller when it targets the source.
func (p sourceTargetedPattern) ByOpponent() sourceTargetedPattern {
	p.opponentOnly = true
	return p
}

// WithController saves the targeting object's controller for this trigger.
func (p sourceTargetedPattern) WithController(binding *PlayerBinding) sourceTargetedPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

// BecomesTargetOfSpell matches a spell that targets the battlefield source.
func BecomesTargetOfSpell(sourceRef) EventPattern { return sourceTargetedBySpellPattern{} }

func (sourceTargetedBySpellPattern) eventPattern() {}
func (sourceTargetedPattern) eventPattern()        {}

func (p sourceTargetedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtBecomesTarget, false, effect).SetConditionData(EventTargetIsSelf{})
	if p.opponentOnly {
		t.AndConditionData(EventPlayerIsNotController{})
	}
	return t
}

func (p sourceTargetedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source becomes the target of a spell or ability", battlefieldSource: true}
	if p.opponentOnly {
		n.Text += " an opponent controls"
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "targeting event: supply one controller binding"
		} else {
			n.bindingWrite = p.controller
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
		}
	}
	return n
}

func (p sourceTargetedPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureController {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.controller, player: event.PlayerID}, Sequence(effects...))}
}

func (sourceTargetedBySpellPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtBecomesTarget, false, effect).SetConditionData(AndTriggerCond{
		Conditions: []TriggerConditionData{EventTargetIsSelf{}, EventFlagIsFalse{}},
	})
}

func (sourceTargetedBySpellPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "source becomes the target of a spell", battlefieldSource: true}
}
