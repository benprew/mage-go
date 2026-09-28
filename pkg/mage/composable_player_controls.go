package mage

import "github.com/google/uuid"

type playerControlsCondition struct {
	player PlayerRef
	filter PermanentFilter
}

// PlayerControls checks the current control of matching battlefield permanents.
func PlayerControls(player PlayerRef, filter PermanentFilter) StateCondition {
	return playerControlsCondition{player: player, filter: filter}
}

func (playerControlsCondition) stateCondition() {}

func (c playerControlsCondition) checkState(g GameReader, ctx StateContext) bool {
	var playerID uuid.UUID
	switch c.player.(type) {
	case controllerRef:
		playerID = ctx.ControllerID
	case activePlayerRef:
		if active := g.ActivePlayerObj(); active != nil {
			playerID = active.PlayerID()
		}
	default:
		return false
	}
	if playerID == uuid.Nil || g.GetPlayer(playerID) == nil {
		return false
	}
	for _, permanent := range g.FilterBattlefield(c.filter) {
		if permanent.ControllerID() == playerID {
			return true
		}
	}
	return false
}

func (c playerControlsCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "player controls matching permanent"}
	switch c.player.(type) {
	case controllerRef, activePlayerRef:
		n.children = []definitionEdge{{"player", c.player}}
	default:
		n.problem = "state condition requires You or ActivePlayer"
	}
	return n
}
