package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type landEntersPattern struct {
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
}

// LandEnters matches each land that enters the battlefield.
func LandEnters() landEntersPattern { return landEntersPattern{} }

// WithController saves the entering land's controller at the entry event.
func (p landEntersPattern) WithController(binding *PlayerBinding) landEntersPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

func (landEntersPattern) eventPattern() {}

func (landEntersPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtZoneChange, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
		EventSourceHasType{Type: TypeLand},
	}})
}

func (p landEntersPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "a land enters", battlefieldSource: true}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "land entry event: supply one controller binding"
		} else {
			n.bindingWrite = p.controller
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
		}
	}
	return n
}

func (p landEntersPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureController {
		return nil
	}
	land := observer.FindPermanent(event.SourceID)
	if land == nil {
		return nil
	}
	return []Effect{&eventPlayerCapture{binding: p.controller, player: land.ControllerID()}}
}
