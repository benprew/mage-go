package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type blockedByPattern struct {
	subtype        string
	filterSubtype  bool
	blocker        *PermanentBinding
	captureBlocker bool
}

// BecomesBlockedBy matches each creature that blocks the source.
func BecomesBlockedBy(sourceRef) blockedByPattern { return blockedByPattern{} }

// OfSubtype limits the blocking creature to one creature subtype.
func (p blockedByPattern) OfSubtype(subtype string) blockedByPattern {
	p.subtype = subtype
	p.filterSubtype = true
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
	}
	return t
}

func (p blockedByPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source becomes blocked by a creature", battlefieldSource: true}
	if p.filterSubtype {
		n.Text = "source becomes blocked by a " + p.subtype
		if p.subtype == "" {
			n.problem = "blocked-by event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "blocked-by event: subtype is not a creature subtype"
		}
	}
	if p.blocker != nil {
		n.bindingWrite = p.blocker
		n.children = append(n.children, definitionEdge{"blocker", &eventPermanentCapture{binding: p.blocker}})
	} else if p.captureBlocker {
		n.problem = "blocked-by event: supply a blocker binding"
	}
	return n
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
