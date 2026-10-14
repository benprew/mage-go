package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PlayerControlsChosenColor checks live subjects against the source entry choice.
// Source departure and phasing cannot supply last-known choices.
func PlayerControlsChosenColor(player PlayerRef, filter PermanentFilter) StateCondition {
	return playerControlsChosenColorCondition{player: player, filter: filter}
}

type playerControlsChosenColorCondition struct {
	player PlayerRef
	filter PermanentFilter
}

func (playerControlsChosenColorCondition) stateCondition() {}
func (c playerControlsChosenColorCondition) checkState(g GameReader, ctx StateContext) bool {
	source := resolveStatePermanent(g, ctx, This)
	if source == nil || source.ChosenColor < core.White || source.ChosenColor > core.Green {
		return false
	}
	return PlayerControls(c.player, And(c.filter, HasColorFilter(source.ChosenColor))).checkState(g, ctx)
}
func (c playerControlsChosenColorCondition) inspectDefinition() definitionNode {
	n := playerControlsCondition(c).inspectDefinition()
	n.Text = "player controls a permanent of the source chosen color"
	n.battlefieldSource = true
	if _, saved := c.player.(savedPlayerRef); saved {
		n.problem = "chosen-color state condition requires You, ActivePlayer, or the source chosen player"
	}
	return n
}

func resolveStatePlayer(g GameReader, ctx StateContext, player PlayerRef) uuid.UUID {
	switch ref := player.(type) {
	case controllerRef:
		return ctx.ControllerID
	case activePlayerRef:
		if active := g.ActivePlayerObj(); active != nil {
			return active.PlayerID()
		}
	case savedPlayerRef:
		return ctx.players[ref.binding]
	case chosenPermanentPlayerRef:
		if _, ok := ref.subject.(sourceRef); !ok {
			return uuid.Nil
		}
		if source := resolveStatePermanent(g, ctx, ref.subject); source != nil {
			return source.ChosenPlayer
		}
	}
	return uuid.Nil
}
