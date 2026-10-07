package mage

import "github.com/google/uuid"

// AttackedDuringLastTurnIncarnation reads the player's last completed turn.
// A returned permanent cannot use an earlier incarnation's history.
func (g *Game) AttackedDuringLastTurnIncarnation(player, permanent, incarnation uuid.UUID) bool {
	return incarnation != uuid.Nil && g.trackers.Turn.attackedLastTurnIncarnation[player][permanent] == incarnation
}

type permanentLastTurnAttackCondition struct {
	subject PermanentRef
	player  PlayerRef
}

// PermanentAttackedDuringLastTurn checks the live subject's last-turn history.
// Phasing keeps history but an absent subject cannot satisfy the condition.
func PermanentAttackedDuringLastTurn(subject PermanentRef, player PlayerRef) StateCondition {
	return permanentLastTurnAttackCondition{subject: subject, player: player}
}

func (permanentLastTurnAttackCondition) stateCondition() {}
func (c permanentLastTurnAttackCondition) checkState(g GameReader, ctx StateContext) bool {
	if nilDefinition(c.subject) || nilDefinition(c.player) {
		return false
	}
	p := resolveStatePermanent(g, ctx, c.subject)
	if p == nil {
		return false
	}
	player := ctx.ControllerID
	switch ref := c.player.(type) {
	case controllerRef:
	case permanentControllerRef:
		owner := resolveStatePermanent(g, ctx, ref.subject)
		if owner == nil {
			return false
		}
		player = owner.ControllerID()
	default:
		return false
	}
	return g.GetPlayer(player) != nil && g.AttackedDuringLastTurnIncarnation(player, p.ID(), p.incarnationID)
}
func (c permanentLastTurnAttackCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "subject attacked during player's last completed turn", battlefieldSource: true, children: []definitionEdge{{"subject", c.subject}, {"player", c.player}}}
	switch c.subject.(type) {
	case sourceRef, attachedPermanentRef:
	default:
		n.problem = "history condition requires This or AttachedPermanent"
	}
	switch ref := c.player.(type) {
	case controllerRef:
	case permanentControllerRef:
		switch ref.subject.(type) {
		case sourceRef, attachedPermanentRef:
		default:
			n.problem = "history player requires a live source or attachment controller"
		}
	default:
		n.problem = "history condition requires You or ControllerOf"
	}
	return n
}
