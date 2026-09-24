package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// TurnStepEventPattern supports ordinary and delayed step-event capture.
type TurnStepEventPattern interface {
	EventPattern
	DelayedStepPattern
}

type turnStepEventPattern struct {
	step   TurnStepPattern
	player *PlayerBinding
}

// TurnStepBegins matches a step pattern and saves the player whose step begins.
// Use FromPlayer in the event body to read that player without a target.
func TurnStepBegins(step TurnStepPattern, player *PlayerBinding) TurnStepEventPattern {
	if nilDefinition(step) {
		panic("step event: supply a pattern")
	}
	return turnStepEventPattern{step, player}
}

func (turnStepEventPattern) eventPattern() {}
func (p turnStepEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return p.step.makeStepTrigger(effect)
}
func (p turnStepEventPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "beginning of " + InspectDefinition(p.step).Text,
		bindingWrite: p.player, children: []definitionEdge{{"step", p.step}, {"player", p.player}}}
}
func (p turnStepEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.player, event.PlayerID}, Sequence(effects...))}
}

func (p turnStepEventPattern) delayedStep(controller uuid.UUID) *DelayedTrigger {
	trigger := p.step.delayedStep(controller)
	trigger.eventPlayerBinding = p.player
	return trigger
}
