package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type landEntersPattern struct {
	subtype             string
	filterSubtype       bool
	duplicateSubtype    bool
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
	actor               *PlayerBinding
	captureActor        bool
	duplicateActor      bool
}

// LandEnters matches each land that enters the battlefield.
func LandEnters() landEntersPattern { return landEntersPattern{} }

// OfSubtype limits the event to an entering land with this land subtype.
func (p landEntersPattern) OfSubtype(subtype string) landEntersPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.subtype, p.filterSubtype = subtype, true
	return p
}

// WithController saves the entering land's controller at the entry event.
func (p landEntersPattern) WithController(binding *PlayerBinding) landEntersPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

// WithActor saves the player who caused the land to enter.
func (p landEntersPattern) WithActor(binding *PlayerBinding) landEntersPattern {
	if p.captureActor {
		p.duplicateActor = true
	}
	p.actor, p.captureActor = binding, true
	return p
}

func (landEntersPattern) eventPattern() {}

func (p landEntersPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	trigger := NewTriggered(EvtZoneChange, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
		EventSourceHasType{Type: TypeLand},
	}})
	if p.filterSubtype && isLandSubtype(p.subtype) {
		trigger.AndConditionData(EventSourceHasSubType{SubType: p.subtype})
	}
	if p.captureActor {
		trigger.AndConditionData(entryActorKnown{})
	}
	return trigger
}

type entryActorKnown struct{}

func (entryActorKnown) CheckTriggerCond(event *GameEvent, observer GameReader, _, _ uuid.UUID) bool {
	return observer.GetPlayer(event.ActorID) != nil
}

func (p landEntersPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "a land enters", battlefieldSource: true}
	if p.filterSubtype {
		n.Text = "a " + p.subtype + " enters"
		if p.duplicateSubtype || !isLandSubtype(p.subtype) {
			n.problem = "land entry event: supply one valid land subtype"
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "land entry event: supply one controller binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.controller)
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
		}
	}
	if p.captureActor {
		if p.actor == nil || p.duplicateActor {
			n.problem = "land entry event: supply one actor binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.actor)
			n.children = append(n.children, definitionEdge{"actor", &eventPlayerCapture{binding: p.actor}})
		}
	}
	if p.captureActor && p.captureController && p.actor == p.controller && p.actor != nil {
		n.problem = "land entry event: actor and controller need different bindings"
	}
	return n
}

func (p landEntersPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureController && !p.captureActor {
		return nil
	}
	var captures []Effect
	if p.captureController {
		land := observer.FindPermanent(event.SourceID)
		if land != nil {
			captures = append(captures, &eventPlayerCapture{binding: p.controller, player: land.ControllerID()})
		}
	}
	if p.captureActor {
		captures = append(captures, &eventPlayerCapture{binding: p.actor, player: event.ActorID})
	}
	return captures
}
