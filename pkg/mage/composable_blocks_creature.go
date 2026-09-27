package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type blocksCreaturePattern struct {
	subtype          string
	filterSubtype    bool
	excludeSubtype   bool
	duplicateFilter  bool
	opponent         *PermanentBinding
	captureOpponent  bool
	duplicateCapture bool
}

// BlocksCreature matches each attacking creature that the source blocks.
func BlocksCreature(sourceRef) blocksCreaturePattern { return blocksCreaturePattern{} }

// OfSubtype requires the attacking creature to have this creature subtype.
func (p blocksCreaturePattern) OfSubtype(subtype string) blocksCreaturePattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes attacking creatures with this creature subtype.
func (p blocksCreaturePattern) WithoutSubtype(subtype string) blocksCreaturePattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

// WithOpponent saves the blocked attacker's battlefield incarnation.
func (p blocksCreaturePattern) WithOpponent(binding *PermanentBinding) blocksCreaturePattern {
	if p.captureOpponent {
		p.duplicateCapture = true
	}
	p.opponent = binding
	p.captureOpponent = true
	return p
}

func (blocksCreaturePattern) eventPattern() {}

func (p blocksCreaturePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtDeclaredBlocker, false, effect).SetConditionData(EventSourceIsSelf{})
	if p.filterSubtype {
		t.AndConditionData(EventTargetHasSubType{SubType: p.subtype})
	}
	if p.excludeSubtype {
		t.AndConditionData(blockerLacksSubtype{subtype: p.subtype})
	}
	return t
}

func (p blocksCreaturePattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source blocks an attacking creature", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.subtype == "" {
			n.problem = "blocks-creature event: supply a creature subtype"
		}
		if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "blocks-creature event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "blocks-creature event: supply one subtype filter"
	}
	if p.opponent != nil {
		n.bindingWrite = p.opponent
		n.children = append(n.children, definitionEdge{"opponent", &eventPermanentCapture{binding: p.opponent}})
	} else if p.captureOpponent {
		n.problem = "blocks-creature event: supply an opponent binding"
	}
	if p.duplicateCapture {
		n.problem = "blocks-creature event: save one opponent"
	}
	return n
}

func (p blocksCreaturePattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureOpponent || p.opponent == nil {
		return nil
	}
	opponent := observer.FindPermanent(event.TargetID)
	if opponent == nil || opponent.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.opponent, value: capturedPermanent{opponent.ID(), opponent.incarnationID}}}
}
