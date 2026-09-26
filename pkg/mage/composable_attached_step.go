package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type attachedControllerUpkeepPattern struct {
	host   *PermanentBinding
	player *PlayerBinding
}

// AttachedControllerUpkeep saves the attachment and its controller when upkeep begins.
// The body reads these subjects with From and FromPlayer without targets.
func AttachedControllerUpkeep(host *PermanentBinding, player *PlayerBinding) EventPattern {
	return attachedControllerUpkeepPattern{host, player}
}
func (attachedControllerUpkeepPattern) eventPattern() {}
func (attachedControllerUpkeepPattern) makeTrigger(body ResolvingEffect) *GenericTriggered {
	return BeginningOfAttachedControllerUpkeepTrigger(body, false)
}
func (p attachedControllerUpkeepPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "attached permanent controller upkeep", battlefieldSource: true,
		children: []definitionEdge{{"host", &eventPermanentCapture{binding: p.host}}, {"player", &eventPlayerCapture{binding: p.player}}}}
}
func (p attachedControllerUpkeepPattern) captureDetectedEvent(event *GameEvent, g GameReader, source uuid.UUID) []Effect {
	var host capturedPermanent
	if aura := g.FindPermanent(source); aura != nil {
		if perm := g.FindPermanent(aura.AttachedTo); perm != nil {
			host = capturedPermanent{perm.ID(), perm.incarnationID}
		}
	}
	return []Effect{&eventPermanentCapture{p.host, host}, &eventPlayerCapture{p.player, event.PlayerID}}
}

type eventPermanentCapture struct {
	binding *PermanentBinding
	value   capturedPermanent
}

func (*eventPermanentCapture) Text() string                 { return "remember the event permanent" }
func (*eventPermanentCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventPermanentCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	ctx.Game.resolution.permanentBindings[e.binding] = e.value
	return nil
}
func (e *eventPermanentCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-permanent-capture", bindingWrite: e.binding, children: []definitionEdge{{"permanent", e.binding}}}
}

func captureDetectedTriggerEvent(ability TriggeredAbility, event *GameEvent, observer GameReader, current *Game, source uuid.UUID) []Effect {
	if trigger, ok := ability.(*GenericTriggered); ok {
		var captures []Effect
		if pattern, ok := trigger.eventDefinition.(interface {
			captureDetectedEvent(*GameEvent, GameReader, uuid.UUID) []Effect
		}); ok {
			captures = pattern.captureDetectedEvent(event, observer, source)
		}
		if pattern, ok := trigger.eventDefinition.(interface {
			captureDestinationEvent(*GameEvent, *Game) []Effect
		}); ok {
			captures = append(captures, pattern.captureDestinationEvent(event, current)...)
		}
		return captures
	}
	return nil
}
