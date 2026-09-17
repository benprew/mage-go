package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// EventPattern describes an event at detection time.
type EventPattern interface {
	eventPattern()
	makeTrigger(ResolvingEffect) *GenericTriggered
}
type sourceEventPattern struct{ dies bool }

func (sourceEventPattern) eventPattern() {}

// Enters matches the source entering the battlefield.
func Enters(sourceRef) EventPattern { return sourceEventPattern{} }

// Dies matches the source leaving the battlefield for the graveyard.
func Dies(sourceRef) EventPattern { return sourceEventPattern{dies: true} }
func (p sourceEventPattern) makeTrigger(e ResolvingEffect) *GenericTriggered {
	if p.dies {
		return DiesTrigger(e, false)
	}
	return EntersBattlefieldTrigger(e, false)
}
func (p sourceEventPattern) inspectDefinition() definitionNode {
	label := "enters"
	if p.dies {
		label = "dies"
	}
	return definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true}
}

// TurnStepPattern describes a turn-step boundary.
type TurnStepPattern interface {
	turnStepPattern()
	makeStepTrigger(ResolvingEffect) *GenericTriggered
}
type stepPattern struct {
	event          EventType
	controllerOnly bool
}

func (stepPattern) turnStepPattern() {}

// YourUpkeep matches the beginning of the controller's upkeep.
var YourUpkeep = stepPattern{EvtUpkeep, true}

// EachUpkeep matches the beginning of each upkeep.
var EachUpkeep = stepPattern{EvtUpkeep, false}

// EachDrawStep matches each draw step without a source-state restriction.
var EachDrawStep = stepPattern{EvtDrawStep, false}

func (p stepPattern) makeStepTrigger(e ResolvingEffect) *GenericTriggered {
	t := NewTriggered(p.event, false, e)
	if p.controllerOnly {
		t.SetConditionData(EventPlayerIsController{})
	}
	return t
}
func (p stepPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "turn-step-pattern", Text: "turn step"}
}

// TriggerOption is a closed option for triggered abilities.
type TriggerOption interface {
	triggerOption()
	applyTrigger(*GenericTriggered)
}

func (targetOptions) triggerOption() {}
func (o targetOptions) applyTrigger(t *GenericTriggered) {
	for _, b := range o.bindings {
		if nilDefinition(b) {
			panic("targets: supply a non-nil binding")
		}
		t.targets = append(t.targets, b.targetSpec())
	}
}

type interveningIfOption struct{ condition StateCondition }

// WithInterveningIf checks the condition at detection and again at resolution.
func WithInterveningIf(condition StateCondition) TriggerOption { return interveningIfOption{condition} }
func (interveningIfOption) triggerOption()                     {}
func (o interveningIfOption) applyTrigger(t *GenericTriggered) {
	if nilDefinition(o.condition) {
		panic("intervening-if: supply a condition")
	}
	if t.interveningDefinition != nil {
		panic("intervening-if: supply only one condition")
	}
	t.interveningDefinition = o.condition
	before := t.Condition
	t.Condition = func(evt *GameEvent, g GameReader, source, controller uuid.UUID) bool {
		return (before == nil || before(evt, g, source, controller)) && o.condition.checkState(g, StateContext{SourceID: source, ControllerID: controller})
	}
	t.effects = []Effect{IfState(o.condition, Sequence(t.effects...))}
}
func applyTriggerOptions(t *GenericTriggered, options []TriggerOption) {
	for _, o := range options {
		if nilDefinition(o) {
			panic("trigger: supply a non-nil option")
		}
		o.applyTrigger(t)
	}
}

// WhenEvent uses the existing event, last-known-information, and pending-trigger paths.
func WhenEvent(pattern EventPattern, effect ResolvingEffect, options ...TriggerOption) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("event: supply a pattern")
	}
	t := pattern.makeTrigger(effect)
	t.eventDefinition = pattern
	applyTriggerOptions(t, options)
	return t
}

// At creates a trigger for a turn-step boundary.
func At(pattern TurnStepPattern, effect ResolvingEffect, options ...TriggerOption) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("step: supply a pattern")
	}
	t := pattern.makeStepTrigger(effect)
	t.stepDefinition = pattern
	applyTriggerOptions(t, options)
	return t
}
