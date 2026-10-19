package mage

import "github.com/google/uuid"

// DefendingPlayerOf reads the defending player of a present attacking visit.
// The two-player combat player remains valid after an attacked permanent leaves.
func DefendingPlayerOf(attacker PermanentRef) PlayerRef {
	return attackerDefendingPlayer{attacker: attacker}
}

type attackerDefendingPlayer struct{ attacker PermanentRef }

func (attackerDefendingPlayer) playerRef()       {}
func (attackerDefendingPlayer) damageRecipient() {}
func (r attackerDefendingPlayer) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	if nilDefinition(r.attacker) {
		return uuid.Nil, false
	}
	p, ok := r.attacker.resolvePermanent(ctx)
	if !ok || !ctx.Game.GetCombat().IsAttackingIncarnation(p.ID(), p.incarnationID) {
		return uuid.Nil, false
	}
	return combatDefendingPlayerID(ctx.Game)
}
func (r attackerDefendingPlayer) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r attackerDefendingPlayer) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "defending player of attacking visit", children: []definitionEdge{{"attacker", r.attacker}}}
}
