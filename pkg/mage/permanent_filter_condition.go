package mage

import "github.com/google/uuid"

type permanentFilterCondition struct {
	subject PermanentRef
	filter  PermanentFilter
}

// PermanentMatchesFilter checks the current source or its attached permanent.
func PermanentMatchesFilter(subject PermanentRef, filter PermanentFilter) StateCondition {
	return permanentFilterCondition{subject: subject, filter: filter}
}

func (permanentFilterCondition) stateCondition() {}

func (c permanentFilterCondition) checkState(g GameReader, ctx StateContext) bool {
	source := g.FindPermanent(ctx.SourceID)
	if source == nil || source.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && source.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	p := source
	switch c.subject.(type) {
	case sourceRef:
	case attachedPermanentRef:
		p = g.FindPermanent(source.AttachedTo)
		if p == nil || source.attachedIncarnation == uuid.Nil || p.incarnationID != source.attachedIncarnation {
			return false
		}
	default:
		return false
	}
	if p.PhasedOut {
		return false
	}
	for _, match := range g.FilterBattlefield(c.filter) {
		if match.ID() == p.ID() && match.incarnationID == p.incarnationID {
			return true
		}
	}
	return false
}

func (c permanentFilterCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "permanent matches " + c.filter.Text(), battlefieldSource: true}
	switch c.subject.(type) {
	case sourceRef, attachedPermanentRef:
	default:
		n.problem = "state condition requires This or AttachedPermanent"
	}
	return n
}
