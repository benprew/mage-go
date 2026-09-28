package mage

import "github.com/google/uuid"

type controlledSinceTurnStartCondition struct{ subject PermanentRef }

// ControlledSinceTurnStart checks continuous control of a battlefield subject.
func ControlledSinceTurnStart(subject PermanentRef) StateCondition {
	return controlledSinceTurnStartCondition{subject: subject}
}

func (controlledSinceTurnStartCondition) stateCondition() {}

func (c controlledSinceTurnStartCondition) checkState(g GameReader, ctx StateContext) bool {
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
	return !p.PhasedOut && ctx.ControllerID != uuid.Nil && p.ControllerID() == ctx.ControllerID && p.ControlledSinceTurnStart(g)
}

func (c controlledSinceTurnStartCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", battlefieldSource: true}
	switch c.subject.(type) {
	case sourceRef:
		n.Text = "source controlled since turn start"
	case attachedPermanentRef:
		n.Text = "attached permanent controlled since turn start"
	default:
		n.problem = "state condition requires This or AttachedPermanent"
	}
	return n
}
