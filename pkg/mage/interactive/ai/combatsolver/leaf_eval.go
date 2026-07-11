package combatsolver

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// combatLeafEval scores a post-combat-damage position for the active player
// (the AI making the decision).
//
// It mirrors eval.WeightedEvaluator but skips the tap-state penalty for the
// AI's own creatures: a tapped attacker on the AI's turn isn't worth less
// than an untapped one — it untaps next turn, and the cost of attacking is
// already captured by any combat damage / death the creature suffered.
//
// The opponent's creatures keep the tap penalty (their tapped permanents
// can't block this turn, which is a real positional cost).
func combatLeafEval(g *mage.Game, playerID uuid.UUID, w eval.Weights) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := float64(me.Life()-opp.Life()) * w.Life

	boardScale := w.Board / 2.0
	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		var v float64
		switch perm.ControllerID() {
		case playerID:
			v = float64(eval.EvalCreatureInGameNoTapPenalty(perm, g)) * boardScale
			score += v
		case oppID:
			v = float64(eval.EvalCreatureInGame(perm, g)) * boardScale
			score -= v
		}
	}

	nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
	for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
		v := float64(evalNonCreatureValue(perm)) * boardScale
		switch perm.ControllerID() {
		case playerID:
			score += v
		case oppID:
			score -= v
		}
	}

	score += float64(len(me.Hand())-len(opp.Hand())) * w.Card

	ownLand := mage.And(mage.IsLand, mage.ControlledBy(playerID))
	score += float64(g.CountBattlefield(ownLand)) * w.Mana
	score += float64(g.CountBattlefield(mage.And(ownLand, mage.IsUntapped))) * w.Tempo

	return int(score)
}

// evalNonCreatureValue is a thin local wrapper that approximates
// eval.evalNonCreaturePermanent (unexported). For combat-decision purposes,
// non-creature non-land permanents don't change during combat, so a
// constant 0 contribution is fine — they cancel between pre/post states.
// We keep this stub for symmetry and to make later expansion (e.g., when
// combat tricks include destroy-permanent effects) trivial.
func evalNonCreatureValue(_ *mage.Permanent) int { return 0 }
