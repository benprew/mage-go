package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type blockerGroupPattern struct {
	subtype         string
	filterSubtype   bool
	excludeSubtype  bool
	duplicateFilter bool
}

// BlocksOrBecomesBlockedBy matches one group of new blocker pairs for the source.
func BlocksOrBecomesBlockedBy(sourceRef) blockerGroupPattern { return blockerGroupPattern{} }

// OfSubtype requires at least one other creature with this creature subtype.
func (p blockerGroupPattern) OfSubtype(subtype string) blockerGroupPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes a creature subtype from the new opposing group.
func (p blockerGroupPattern) WithoutSubtype(subtype string) blockerGroupPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

func (blockerGroupPattern) eventPattern() {}

func (p blockerGroupPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtBlockerGroup, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventSourceIsSelf{}, blockerGroupMatches{subtype: p.subtype, filterSubtype: p.filterSubtype, excludeSubtype: p.excludeSubtype},
	}})
}

func (p blockerGroupPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source blocks or becomes blocked by one or more creatures", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.excludeSubtype {
			n.Text = "source blocks or becomes blocked by one or more creatures without " + p.subtype
		} else {
			n.Text = "source blocks or becomes blocked by one or more " + p.subtype + " creatures"
		}
		if p.subtype == "" {
			n.problem = "blocker group event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "blocker group event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "blocker group event: supply one subtype filter"
	}
	return n
}

type blockerGroupMatches struct {
	subtype        string
	filterSubtype  bool
	excludeSubtype bool
}

func (c blockerGroupMatches) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	for _, id := range event.RelatedIDs {
		other := reader.FindPermanent(id)
		if other != nil && !other.PhasedOut && other.HasType(TypeCreature) &&
			(!c.filterSubtype || other.HasSubType(c.subtype)) && (!c.excludeSubtype || !other.HasSubType(c.subtype)) {
			return true
		}
	}
	return false
}
