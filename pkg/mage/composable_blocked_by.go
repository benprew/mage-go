package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type blockedByPattern struct {
	subtype         string
	filterSubtype   bool
	excludeSubtype  bool
	duplicateFilter bool
	blocker         *PermanentBinding
	captureBlocker  bool
}

// BecomesBlockedBy matches each creature that blocks the source.
func BecomesBlockedBy(sourceRef) blockedByPattern { return blockedByPattern{} }

// OfSubtype limits the blocking creature to one creature subtype.
func (p blockedByPattern) OfSubtype(subtype string) blockedByPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes a blocker with this creature subtype at the block event.
func (p blockedByPattern) WithoutSubtype(subtype string) blockedByPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

// WithBlocker saves the blocking creature's battlefield incarnation.
func (p blockedByPattern) WithBlocker(blocker *PermanentBinding) blockedByPattern {
	p.blocker = blocker
	p.captureBlocker = true
	return p
}

func (blockedByPattern) eventPattern() {}

func (p blockedByPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtCreatureBlockedBy, false, effect).SetConditionData(EventSourceIsSelf{})
	if p.filterSubtype {
		t.AndConditionData(EventTargetHasSubType{SubType: p.subtype})
	} else if p.excludeSubtype {
		t.AndConditionData(blockerLacksSubtype{subtype: p.subtype})
	}
	return t
}

func (p blockedByPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source becomes blocked by a creature", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.excludeSubtype {
			n.Text = "source becomes blocked by a creature without " + p.subtype
		} else {
			n.Text = "source becomes blocked by a " + p.subtype
		}
		if p.subtype == "" {
			n.problem = "blocked-by event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "blocked-by event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "blocked-by event: supply one subtype filter"
	}
	if p.blocker != nil {
		n.bindingWrite = p.blocker
		n.children = append(n.children, definitionEdge{"blocker", &eventPermanentCapture{binding: p.blocker}})
	} else if p.captureBlocker {
		n.problem = "blocked-by event: supply a blocker binding"
	}
	return n
}

type blockerLacksSubtype struct{ subtype string }

func (c blockerLacksSubtype) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	blocker := reader.FindPermanent(event.TargetID)
	return blocker != nil && !blocker.PhasedOut && blocker.HasType(TypeCreature) && !blocker.HasSubType(c.subtype)
}

func (p blockedByPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureBlocker || p.blocker == nil {
		return nil
	}
	blocker := observer.FindPermanent(event.TargetID)
	if blocker == nil || blocker.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.blocker, value: capturedPermanent{blocker.ID(), blocker.incarnationID}}}
}
