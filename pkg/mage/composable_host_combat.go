package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type attachedCombatPattern struct {
	host      *PermanentBinding
	attack    bool
	block     bool
	unblocked bool
}

// AttachedAttacks saves the attached creature when it attacks.
func AttachedAttacks(host *PermanentBinding) EventPattern {
	return attachedCombatPattern{host: host, attack: true}
}

// AttachedBlocks saves the attached creature once when it blocks.
func AttachedBlocks(host *PermanentBinding) EventPattern {
	return attachedCombatPattern{host: host, block: true}
}

// AttachedAttacksUnblocked saves the attached attacker after blocker declaration.
func AttachedAttacksUnblocked(host *PermanentBinding) EventPattern {
	return attachedCombatPattern{host: host, unblocked: true}
}

// AttachedAttacksOrBlocks saves the attached creature when it attacks or blocks.
func AttachedAttacksOrBlocks(host *PermanentBinding) EventPattern {
	return attachedCombatPattern{host: host, attack: true, block: true}
}

func (attachedCombatPattern) eventPattern() {}

func (p attachedCombatPattern) matchesEventType(kind EventType) bool {
	return p.attack && kind == EvtDeclaredAttacker || p.block && kind == EvtDeclaredBlocker || p.unblocked && kind == EvtAttacksUnblocked
}

func (p attachedCombatPattern) makeTrigger(body ResolvingEffect) *GenericTriggered {
	kind := EvtDeclaredAttacker
	if p.unblocked {
		kind = EvtAttacksUnblocked
	} else if !p.attack {
		kind = EvtDeclaredBlocker
	}
	return NewTriggered(kind, false, body).SetConditionData(attachedCombatCondition{})
}

type attachedCombatCondition struct{}

func (attachedCombatCondition) CheckTriggerCond(event *GameEvent, g GameReader, source, controller uuid.UUID) bool {
	if event.Type == EvtDeclaredBlocker && !event.Flag {
		return false
	}
	host := g.FindPermanent(event.SourceID)
	return host != nil && !host.PhasedOut && host.HasType(TypeCreature) &&
		(event.SourceIncarnation == uuid.Nil || event.SourceIncarnation == host.incarnationID) &&
		(EventSourceIsAttachedTo{}).CheckTriggerCond(event, g, source, controller)
}

func (p attachedCombatPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "attached creature attacks or blocks", battlefieldSource: true, bindingWrite: p.host,
		children: []definitionEdge{{"host", &eventPermanentCapture{binding: p.host}}}}
}

func (p attachedCombatPattern) captureDetectedEvent(event *GameEvent, g GameReader, _ uuid.UUID) []Effect {
	var host capturedPermanent
	if permanent := g.FindPermanent(event.SourceID); permanent != nil {
		host = capturedPermanent{permanent.ID(), permanent.incarnationID}
	}
	return []Effect{&eventPermanentCapture{binding: p.host, value: host}}
}
