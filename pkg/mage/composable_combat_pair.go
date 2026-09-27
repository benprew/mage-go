package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type combatPairPattern struct {
	subtype          string
	filterSubtype    bool
	excludeSubtype   bool
	duplicateFilter  bool
	opponent         *PermanentBinding
	captureOpponent  bool
	duplicateCapture bool
}

// BlocksOrBecomesBlockedByCreature matches each new opposing creature in combat.
func BlocksOrBecomesBlockedByCreature(sourceRef) combatPairPattern { return combatPairPattern{} }

// OfSubtype requires the opposing creature to have this creature subtype.
func (p combatPairPattern) OfSubtype(subtype string) combatPairPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes an opposing creature with this creature subtype.
func (p combatPairPattern) WithoutSubtype(subtype string) combatPairPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

// WithOpponent saves the opposing creature's battlefield incarnation.
func (p combatPairPattern) WithOpponent(binding *PermanentBinding) combatPairPattern {
	if p.captureOpponent {
		p.duplicateCapture = true
	}
	p.opponent = binding
	p.captureOpponent = true
	return p
}

func (combatPairPattern) eventPattern() {}

func (p combatPairPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtCombatPair, false, effect).SetConditionData(EventSourceIsSelf{})
	if p.filterSubtype {
		t.AndConditionData(EventTargetHasSubType{SubType: p.subtype})
	} else if p.excludeSubtype {
		t.AndConditionData(blockerLacksSubtype{subtype: p.subtype})
	}
	return t
}

func (p combatPairPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source blocks or becomes blocked by an opposing creature", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.subtype == "" {
			n.problem = "combat-pair event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "combat-pair event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "combat-pair event: supply one subtype filter"
	}
	if p.opponent != nil {
		n.bindingWrite = p.opponent
		n.children = append(n.children, definitionEdge{"opponent", &eventPermanentCapture{binding: p.opponent}})
	} else if p.captureOpponent {
		n.problem = "combat-pair event: supply an opponent binding"
	}
	if p.duplicateCapture {
		n.problem = "combat-pair event: save one opponent"
	}
	return n
}

func (p combatPairPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureOpponent || p.opponent == nil {
		return nil
	}
	opponent := observer.FindPermanent(event.TargetID)
	if opponent == nil || opponent.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.opponent, value: capturedPermanent{opponent.ID(), opponent.incarnationID}}}
}
