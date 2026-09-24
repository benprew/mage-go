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

type creatureDiesPattern struct{ controllerOnly bool }

// CreatureDies matches any creature moving from the battlefield to a graveyard.
// It includes the source and uses the creature's type before it leaves.
func CreatureDies() EventPattern { return creatureDiesPattern{} }

// CreatureYouControlDies matches a creature controlled by you before it dies.
func CreatureYouControlDies() EventPattern { return creatureDiesPattern{controllerOnly: true} }

func (creatureDiesPattern) eventPattern() {}
func (p creatureDiesPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtZoneChange, false, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
			EventSourceWasOfType{Type: TypeCreature},
		}})
	if p.controllerOnly {
		t.AndConditionData(EventPlayerIsController{})
	}
	return t
}
func (p creatureDiesPattern) inspectDefinition() definitionNode {
	text := "a creature dies"
	if p.controllerOnly {
		text = "a creature you control dies"
	}
	return definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true}
}

// TurnStepPattern describes a turn-step boundary.
type TurnStepPattern interface {
	turnStepPattern()
	makeStepTrigger(ResolvingEffect) *GenericTriggered
	delayedStep(uuid.UUID) *DelayedTrigger
}
type stepPattern struct {
	event          EventType
	controllerOnly bool
}

func (stepPattern) turnStepPattern() {}
func (p stepPattern) delayedStep(controller uuid.UUID) *DelayedTrigger {
	t := &DelayedTrigger{EventType: p.event}
	if p.controllerOnly {
		t.MatchPlayerID = controller
	}
	return t
}

// EachEndStep matches the beginning of each end step.
var EachEndStep = stepPattern{EvtEndStep, false}

// YourEndStep matches the beginning of the controller's end step.
var YourEndStep = stepPattern{EvtEndStep, true}

// EachEndOfCombat matches the beginning of each end-of-combat step.
var EachEndOfCombat = stepPattern{EvtEndOfCombat, false}

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
	label := p.event.String()
	if p.controllerOnly {
		label = "your " + label
	}
	return definitionNode{Role: "turn-step-pattern", Text: label}
}

type mainPhasePattern struct{ first bool }

// YourFirstMainPhase matches the beginning of your first main phase.
var YourFirstMainPhase = mainPhasePattern{first: true}

// YourPostcombatMainPhase matches the beginning of each of your postcombat main phases.
var YourPostcombatMainPhase = mainPhasePattern{}

func (mainPhasePattern) turnStepPattern() {}
func (p mainPhasePattern) makeStepTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtMainPhase, false, effect).SetConditionData(EventPlayerIsController{})
	if p.first {
		t.AndConditionData(EventFlagIsTrue{})
	} else {
		t.AndConditionData(EventFlagIsFalse{})
	}
	return t
}
func (p mainPhasePattern) delayedStep(controller uuid.UUID) *DelayedTrigger {
	return &DelayedTrigger{EventType: EvtMainPhase, MatchPlayerID: controller, MatchFlag: p.first, matchFlagFalse: !p.first}
}
func (p mainPhasePattern) inspectDefinition() definitionNode {
	text := "your postcombat main phase"
	if p.first {
		text = "your first main phase"
	}
	return definitionNode{Role: "turn-step-pattern", Text: text}
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
func WithInterveningIf(condition StateCondition) interveningIfOption {
	return interveningIfOption{condition}
}
func (interveningIfOption) triggerOption() {}
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

// WithTargets adds typed targets to a legacy event trigger.
func (t *GenericTriggered) WithTargets(bindings ...TargetBinding) *GenericTriggered {
	WithTargets(bindings...).applyTrigger(t)
	return t
}
