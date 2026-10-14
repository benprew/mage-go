package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CombatDefendingPlayer selects the nonactive player during two-player combat.
func CombatDefendingPlayer() PlayerRef { return combatDefendingPlayerRef{} }

type combatDefendingPlayerRef struct{}

func (combatDefendingPlayerRef) playerRef()       {}
func (combatDefendingPlayerRef) damageRecipient() {}
func (r combatDefendingPlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (combatDefendingPlayerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	return combatDefendingPlayerID(ctx.Game)
}

func combatDefendingPlayerID(g GameReader) (uuid.UUID, bool) {
	step := g.GetStep()
	players := g.AllPlayers()
	if step < core.BeginCombat || step > core.EndCombat || len(players) != 2 {
		return uuid.Nil, false
	}
	active := g.ActivePlayerObj()
	if active == nil {
		return uuid.Nil, false
	}
	for _, player := range players {
		if player.PlayerID() != active.PlayerID() {
			return player.PlayerID(), true
		}
	}
	return uuid.Nil, false
}
func (combatDefendingPlayerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "two-player combat defending player"}
}
