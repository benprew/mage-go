package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PlayerControlsNone requires a known player before it checks absent membership.
func PlayerControlsNone(player PlayerRef, filter PermanentFilter) StateCondition {
	return playerControlsNoneCondition{player: player, filter: filter}
}

type playerControlsNoneCondition struct {
	player PlayerRef
	filter PermanentFilter
}

func (playerControlsNoneCondition) stateCondition() {}
func (c playerControlsNoneCondition) checkState(g GameReader, ctx StateContext) bool {
	playerID := resolveStatePlayer(g, ctx, c.player)
	if playerID == uuid.Nil || g.GetPlayer(playerID) == nil {
		return false
	}
	return !PlayerControls(c.player, c.filter).checkState(g, ctx)
}
func (c playerControlsNoneCondition) inspectDefinition() definitionNode {
	n := playerControlsCondition(c).inspectDefinition()
	n.Text = "player controls no matching permanents"
	return n
}

// PlayerControlsNoneOfChosenColor requires a live source choice for absence.
// Missing information cannot satisfy the condition.
func PlayerControlsNoneOfChosenColor(player PlayerRef, filter PermanentFilter) StateCondition {
	return playerControlsNoneOfChosenColorCondition{player: player, filter: filter}
}

type playerControlsNoneOfChosenColorCondition struct {
	player PlayerRef
	filter PermanentFilter
}

func (playerControlsNoneOfChosenColorCondition) stateCondition() {}
func (c playerControlsNoneOfChosenColorCondition) checkState(g GameReader, ctx StateContext) bool {
	source := resolveStatePermanent(g, ctx, This)
	if source == nil || source.ChosenColor < core.White || source.ChosenColor > core.Green {
		return false
	}
	return PlayerControlsNone(c.player, And(c.filter, HasColorFilter(source.ChosenColor))).checkState(g, ctx)
}
func (c playerControlsNoneOfChosenColorCondition) inspectDefinition() definitionNode {
	n := playerControlsChosenColorCondition(c).inspectDefinition()
	n.Text = "player controls no permanents of the source chosen color"
	return n
}
