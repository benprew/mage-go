package mage

import "github.com/google/uuid"

type permanentPlayerCondition struct {
	subject PermanentRef
	owner   bool
}

// PlayerIsPermanentOwner checks the player against a live source or attachment.
func PlayerIsPermanentOwner(subject PermanentRef) StateCondition {
	return permanentPlayerCondition{subject: subject, owner: true}
}

// PlayerIsPermanentController checks the player against a live source or attachment.
func PlayerIsPermanentController(subject PermanentRef) StateCondition {
	return permanentPlayerCondition{subject: subject}
}

func (permanentPlayerCondition) stateCondition() {}

func (c permanentPlayerCondition) checkState(g GameReader, ctx StateContext) bool {
	if ctx.ControllerID == uuid.Nil || g.GetPlayer(ctx.ControllerID) == nil {
		return false
	}
	p := resolveStatePermanent(g, ctx, c.subject)
	if p == nil {
		return false
	}
	if c.owner {
		return p.Card.Owner() == ctx.ControllerID
	}
	return p.ControllerID() == ctx.ControllerID
}

func (c permanentPlayerCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "player is permanent controller", battlefieldSource: true}
	if c.owner {
		n.Text = "player is permanent owner"
	}
	switch c.subject.(type) {
	case sourceRef, attachedPermanentRef:
	default:
		n.problem = "player condition requires This or AttachedPermanent"
	}
	return n
}
