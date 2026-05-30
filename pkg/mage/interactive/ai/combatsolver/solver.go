package combatsolver

import (
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// SolveAttack returns the attacker subset playerID should declare in the
// current DeclareAttackers step, optimised against the opponent's best
// blocking response.
//
// Algorithm: staged minimax over (attackers × blocks). Per-subset blocking
// scores are computed from the AI's perspective; the opponent picks the
// blocking response that minimises the AI's score (worst-case-for-AI). The
// L1 attacker pick maximises that worst case (max-min).
//
// Phase 2 scope: no L3 trick / activated-ability response yet (HeldTricks
// remains empty). Two damage-step model and on-board ability enumeration
// land in subsequent commits.
func SolveAttack(g *mage.Game, playerID uuid.UUID, opts Options) Result {
	if g == nil {
		return Result{}
	}
	opp := g.GetOpponent(playerID)
	if opp == nil {
		return Result{}
	}
	oppID := opp.PlayerID()

	atkSets := enumerateAttackerSets(g, playerID)
	if len(atkSets) == 0 {
		return Result{}
	}

	weights := opts.Profile.Weights
	if isZeroWeights(weights) {
		weights = defaultLeafWeights
	}
	leaf := func(g *mage.Game, pid uuid.UUID) int {
		return combatLeafEval(g, pid, weights)
	}

	var (
		best        Result
		bestScore   = math.MinInt
		nodes       int
		hitDeadline bool
	)

	for setIdx, atkSet := range atkSets {
		if deadlinePassed(opts.Deadline) {
			hitDeadline = true
			break
		}

		score, ok := worstBlockingResponse(g, playerID, oppID, atkSet, leaf, opts.Deadline, &nodes)
		if !ok {
			hitDeadline = true
			break
		}

		if setIdx == 0 || score > bestScore {
			bestScore = score
			best.Attackers = atkSet
			best.Score = score
		}
	}

	best.Nodes = nodes
	best.DeadlineHit = hitDeadline
	return best
}

// SolveDefense returns the block assignments playerID should declare in the
// current DeclareBlockers step, given the attackers already declared by the
// opponent. It picks the blocker assignment that maximises the AI's leaf
// evaluation after combat damage resolves.
//
// Phase 2 scope: no L2 trick / activated-ability response yet.
func SolveDefense(g *mage.Game, playerID uuid.UUID, opts Options) Result {
	if g == nil {
		return Result{}
	}

	blockSets := enumerateBlockerSets(g, playerID)
	if len(blockSets) == 0 {
		return Result{}
	}

	weights := opts.Profile.Weights
	if isZeroWeights(weights) {
		weights = defaultLeafWeights
	}
	leaf := func(g *mage.Game, pid uuid.UUID) int {
		return combatLeafEval(g, pid, weights)
	}

	var (
		best        Result
		bestScore   = math.MinInt
		nodes       int
		hitDeadline bool
	)

	for setIdx, blockSet := range blockSets {
		if deadlinePassed(opts.Deadline) {
			hitDeadline = true
			break
		}

		clone := g.Clone()
		if clone == nil {
			continue
		}
		clone.ExecuteBlockers(blockSet)
		applyCombatSelfPump(clone, playerID, blockSet)
		clone.ExecuteCombatDamage()
		clone.CheckStateBasedActions()
		score := leaf(clone, playerID)
		nodes++

		if setIdx == 0 || score > bestScore {
			bestScore = score
			best.Blocks = blockSet
			best.Score = score
		}
	}

	best.Nodes = nodes
	best.DeadlineHit = hitDeadline
	return best
}

// worstBlockingResponse returns the AI's score under the opponent's best
// blocking response to the given attacker set. Bool false means the deadline
// hit before a response could be evaluated.
func worstBlockingResponse(g *mage.Game, playerID, oppID uuid.UUID,
	attackers []uuid.UUID, leaf eval.StateEvaluator, deadline time.Time, nodes *int) (int, bool) {

	postAttack := g.Clone()
	if postAttack == nil {
		return 0, false
	}
	postAttack.ExecuteAttackers(playerID, attackers)
	postAttack.PutTriggersOnStack()
	postAttack.CheckStateBasedActions()
	postAttack.ResolveStack()

	blockSets := enumerateBlockerSets(postAttack, oppID)
	if len(blockSets) == 0 {
		// No blocks possible — resolve damage and evaluate.
		clone := postAttack.Clone()
		clone.ExecuteCombatDamage()
		clone.CheckStateBasedActions()
		*nodes++
		return leaf(clone, playerID), true
	}

	worst := math.MaxInt
	for _, blockSet := range blockSets {
		if deadlinePassed(deadline) {
			if worst == math.MaxInt {
				return 0, false
			}
			return worst, true
		}
		clone := postAttack.Clone()
		clone.ExecuteBlockers(blockSet)
		applyCombatSelfPump(clone, oppID, blockSet)
		clone.ExecuteCombatDamage()
		clone.CheckStateBasedActions()
		score := leaf(clone, playerID)
		*nodes++
		if score < worst {
			worst = score
		}
	}
	return worst, true
}

// defaultLeafWeights is used when the caller passes a zero-valued Profile.
// Mirrors the historical DefaultEvaluator constants (Life=3, Board=2 → boardScale=1).
var defaultLeafWeights = eval.Weights{Life: 3, Board: 2, Card: 2, Mana: 1, Tempo: 0}

func deadlinePassed(deadline time.Time) bool {
	if deadline.IsZero() {
		return false
	}
	return time.Now().After(deadline)
}

func isZeroWeights(w eval.Weights) bool {
	return w.Life == 0 && w.Board == 0 && w.Card == 0 && w.Mana == 0 && w.Tempo == 0
}
