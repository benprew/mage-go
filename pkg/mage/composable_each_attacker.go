package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type eachControlledAttackerPattern struct {
	attacker          *PermanentBinding
	defender          *PlayerBinding
	captureAttacker   bool
	captureDefender   bool
	duplicateAttacker bool
	duplicateDefender bool
}

// EachCreatureYouControlAttacks matches each declared attacker you control.
func EachCreatureYouControlAttacks() eachControlledAttackerPattern {
	return eachControlledAttackerPattern{}
}

// WithAttacker saves the attacking permanent's battlefield incarnation.
func (p eachControlledAttackerPattern) WithAttacker(binding *PermanentBinding) eachControlledAttackerPattern {
	if p.captureAttacker {
		p.duplicateAttacker = true
	}
	p.attacker, p.captureAttacker = binding, true
	return p
}

// WithDefendingPlayer saves the attacked player in this combat model.
func (p eachControlledAttackerPattern) WithDefendingPlayer(binding *PlayerBinding) eachControlledAttackerPattern {
	if p.captureDefender {
		p.duplicateDefender = true
	}
	p.defender, p.captureDefender = binding, true
	return p
}

func (eachControlledAttackerPattern) eventPattern() {}

func (eachControlledAttackerPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtDeclaredAttacker, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventPlayerIsController{}, declaredAttackerPresent{},
	}})
}

type declaredAttackerPresent struct{}

func (declaredAttackerPresent) CheckTriggerCond(event *GameEvent, reader GameReader, _, controller uuid.UUID) bool {
	permanent := reader.FindPermanent(event.SourceID)
	return permanent != nil && !permanent.PhasedOut && permanent.ControllerID() == controller && permanent.HasType(TypeCreature)
}

func (p eachControlledAttackerPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "each creature you control attacks", battlefieldSource: true}
	if p.captureAttacker {
		if p.attacker == nil || p.duplicateAttacker {
			n.problem = "attack event: supply one attacker binding"
		} else {
			n.children = append(n.children, definitionEdge{"attacker", &eventPermanentCapture{binding: p.attacker}})
			n.bindingWrites = append(n.bindingWrites, p.attacker)
		}
	}
	if p.captureDefender {
		if p.defender == nil || p.duplicateDefender {
			n.problem = "attack event: supply one defending player binding"
		} else {
			n.children = append(n.children, definitionEdge{"defender", &eventPlayerCapture{binding: p.defender}})
			n.bindingWrites = append(n.bindingWrites, p.defender)
		}
	}
	return n
}

func (p eachControlledAttackerPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.captureAttacker {
		if permanent := observer.FindPermanent(event.SourceID); permanent != nil && !permanent.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.attacker, value: capturedPermanent{permanent.ID(), permanent.incarnationID}})
		}
	}
	if p.captureDefender {
		captures = append(captures, &eventPlayerCapture{binding: p.defender, player: event.TargetID})
	}
	return captures
}
