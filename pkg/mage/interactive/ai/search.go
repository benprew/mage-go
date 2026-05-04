package ai

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai/combatsolver"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// DebugSearchStats enables one-line per-decision logging of search telemetry
// (nodes visited, depth reached, elapsed time vs. budget, chosen move).
var DebugSearchStats bool

// SearchConfig controls the search parameters.
type SearchConfig struct {
	MaxDepth  int
	MaxNodes  int
	TimeLimit time.Duration
}

// DefaultSearchConfig returns the default search configuration.
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		MaxDepth:  10,
		MaxNodes:  20000,
		TimeLimit: 1500 * time.Millisecond,
	}
}

// SearchStrategy implements AIStrategy using minimax search with alpha-beta pruning.
// It uses Game.Clone() for accurate game state simulation and searches across
// all decision points: priority actions, attacker declarations, and blocker assignments.
//
// The personality's decision weights (Aggression, TargetFace, etc.) influence both
// the leaf-node evaluator (via Aggression) and move ordering (via the other weights),
// so that the same minimax search produces different play styles per personality.
//
// Optimizations:
//   - Iterative deepening with PV (principal variation) move ordering
//   - Null-move pruning: skip opponent's turn for fast bound estimation
//   - Late move reductions: reduce depth for low-heuristic moves
//   - History heuristic: track which move types cause cutoffs for better ordering
type SearchStrategy struct {
	Config      SearchConfig
	Evaluator   eval.StateEvaluator
	Fallback    *HeuristicStrategy
	Personality WeightedPersonality

	// history tracks cutoff counts per move key for move ordering.
	// Persists across calls within the same strategy instance.
	history map[string]int

	// Transposition table and the Zobrist tables used to key it. When either
	// is nil the TT path is skipped entirely — the old minimax behavior.
	tt      *TranspositionTable
	zobrist *ZobristTables

	// TT telemetry counters. Monotonic; callers may snapshot before/after a
	// top-level decision to derive per-move hit rates.
	TTHits   uint64
	TTStores uint64
	// LastNodes is the node count of the most recent PriorityAction search,
	// summed across iterative-deepening depths. Useful for A/B measuring
	// the effect of pruning optimizations.
	LastNodes uint64
}

const (
	minScore = -1000000
	maxScore = 1000000
)

// maxMoveChain limits sequential moves per player within a single search ply
// to prevent combinatorial explosion when exploring multi-spell turns.
const maxMoveChain = 4

// nullMoveReduction is the depth reduction for null-move pruning.
const nullMoveReduction = 2

// lmrMinDepth is the minimum depth at which late move reductions apply.
const lmrMinDepth = 3

// lmrMoveThreshold is the move index after which LMR kicks in.
const lmrMoveThreshold = 3

// maxQuiescenceDepth is the maximum number of extra plies searched
// by quiescence search when the position is tactical at depth 0.
const maxQuiescenceDepth = 2

// aspirationDelta is the half-width of the aspiration window around the
// previous depth's score. A tighter window causes more alpha-beta cutoffs,
// typically giving 2-3x speedup. If a search fails outside the window, it
// is re-searched with progressively wider bounds.
const aspirationDelta = 50

// moveKey returns a string key for history heuristic tracking.
// Includes the first target ID when available to differentiate moves
// like "Bolt targeting opponent" from "Bolt targeting creature".
func moveKey(m *Move) string {
	if m.Type == interactive.ActionPass {
		return "pass"
	}
	key := m.CardName + ":" + string(rune(m.Type))
	if len(m.Targets) > 0 {
		key += ":" + m.Targets[0].String()
	}
	return key
}

// adjustMoveHeuristics applies personality decision weights to move ordering.
// This biases the search toward exploring personality-appropriate moves first,
// improving pruning efficiency and making the AI's play style match its personality
// even when all moves are ultimately evaluated by minimax.
func (s *SearchStrategy) adjustMoveHeuristics(moves []Move, g *mage.Game, playerID uuid.UUID, mainPhase bool) {
	wp := s.Personality
	for i := range moves {
		m := &moves[i]
		if m.Type == interactive.ActionPass {
			continue
		}

		// TargetFace: boost spells that target the opponent directly.
		if wp.TargetFace > 0.1 && len(m.Targets) > 0 {
			opp := g.GetOpponent(playerID)
			if opp != nil && m.Targets[0] == opp.PlayerID() {
				m.heuristic += int(wp.TargetFace * 8)
			}
		}

		// CurvePreference: 1.0 = prefer expensive spells, 0.0 = prefer cheap.
		// Adjust heuristic based on whether the move is a spell cast.
		if m.Type == interactive.ActionCastSpell && wp.CurvePreference > 0.1 {
			// Find the card's CMC to bias ordering.
			for _, c := range g.GetPlayer(playerID).Hand() {
				if c.ID() == m.CardID {
					cmc := c.ManaCost().CMC()
					// CurvePreference=1.0 → +3 per CMC; CurvePreference=0.0 → -1 per CMC
					bias := (wp.CurvePreference - 0.5) * 2.0 // maps [0,1] to [-1,1]
					m.heuristic += int(bias * float64(cmc))
					break
				}
			}
		}

		// HoldInstants: penalize casting instants during main phase.
		// High HoldInstants means the AI prefers to hold instants for the opponent's turn.
		if wp.HoldInstants > 0.1 && mainPhase && m.Type == interactive.ActionCastSpell {
			for _, c := range g.GetPlayer(playerID).Hand() {
				if c.ID() == m.CardID && c.HasType(core.TypeInstant) {
					m.heuristic -= int(wp.HoldInstants * 6)
					break
				}
			}
		}
	}
}

// ── Priority Action Search ──────────────────────────────────────────────────

func (s *SearchStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	moves := GeneratePriorityMoves(g, p, landsPlayed, mainPhase)
	if len(moves) <= 1 {
		return interactive.PriorityAction{Type: interactive.ActionPass}
	}

	// Apply personality-driven move ordering adjustments and re-sort.
	s.adjustMoveHeuristics(moves, g, p.PlayerID(), mainPhase)
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].heuristic > moves[j].heuristic
	})

	if s.history == nil {
		s.history = make(map[string]int)
	}

	start := time.Now()
	deadline := start.Add(s.Config.TimeLimit)

	// Iterative deepening with PV move ordering: the best move from the
	// previous depth is searched first at the next depth.
	var bestMove *Move
	bestScore := minScore
	var pvIndex int // index of PV move for next iteration
	var totalNodes uint64
	var lastCompletedDepth int

	for depth := 1; depth <= s.Config.MaxDepth; depth++ {
		// Search PV move first (from previous iteration).
		if depth > 1 && pvIndex < len(moves) && pvIndex > 0 {
			moves[0], moves[pvIndex] = moves[pvIndex], moves[0]
			pvIndex = 0
		}

		// Apply history heuristic ordering to non-PV moves.
		if depth > 1 && len(moves) > 2 {
			s.sortByHistory(moves[1:])
		}

		// Aspiration windows: at depth > 1, search with a narrow window
		// centered on the previous depth's score. If the result falls
		// outside the window (fail-low or fail-high), re-search with
		// progressively wider bounds.
		alpha := minScore
		beta := maxScore
		if depth > 1 && bestScore > minScore && bestScore < maxScore {
			alpha = bestScore - aspirationDelta
			beta = bestScore + aspirationDelta
		}

		nodes := 0
		depthBestScore, depthBestMove, depthBestIdx := s.searchRoot(
			g, p, moves, depth, alpha, beta, landsPlayed, &nodes, deadline)

		// Aspiration window re-searches when the score falls outside.
		if depth > 1 && depthBestMove != nil && (alpha > minScore || beta < maxScore) {
			if depthBestScore <= alpha {
				// Fail-low: the true score is lower than our window.
				// Re-search with open lower bound but keep upper bound.
				nodes2 := 0
				score2, move2, idx2 := s.searchRoot(
					g, p, moves, depth, minScore, beta, landsPlayed, &nodes2, deadline)
				nodes += nodes2
				if move2 != nil {
					depthBestScore, depthBestMove, depthBestIdx = score2, move2, idx2
				}
				// If still failing, re-search with full window.
				if depthBestScore >= beta {
					nodes3 := 0
					score3, move3, idx3 := s.searchRoot(
						g, p, moves, depth, minScore, maxScore, landsPlayed, &nodes3, deadline)
					nodes += nodes3
					if move3 != nil {
						depthBestScore, depthBestMove, depthBestIdx = score3, move3, idx3
					}
				}
			} else if depthBestScore >= beta {
				// Fail-high: the true score is higher than our window.
				// Re-search with open upper bound but keep lower bound.
				nodes2 := 0
				score2, move2, idx2 := s.searchRoot(
					g, p, moves, depth, alpha, maxScore, landsPlayed, &nodes2, deadline)
				nodes += nodes2
				if move2 != nil {
					depthBestScore, depthBestMove, depthBestIdx = score2, move2, idx2
				}
				// If still failing, re-search with full window.
				if depthBestScore <= alpha {
					nodes3 := 0
					score3, move3, idx3 := s.searchRoot(
						g, p, moves, depth, minScore, maxScore, landsPlayed, &nodes3, deadline)
					nodes += nodes3
					if move3 != nil {
						depthBestScore, depthBestMove, depthBestIdx = score3, move3, idx3
					}
				}
			}
		}

		// Update overall best if this depth completed or found something better.
		if depthBestMove != nil && depthBestScore > bestScore {
			bestScore = depthBestScore
			bestMove = depthBestMove
			pvIndex = depthBestIdx
		}

		totalNodes += uint64(nodes)
		lastCompletedDepth = depth

		if time.Now().After(deadline) {
			break
		}
	}

	s.LastNodes = totalNodes

	action := interactive.PriorityAction{Type: interactive.ActionPass}
	usedFallback := false
	switch {
	case bestMove == nil:
		action = s.Fallback.PriorityAction(p, g, landsPlayed, mainPhase)
		usedFallback = true
	case bestScore < s.eval(g, p.PlayerID()) && bestMove.Type != interactive.ActionPlayLand && !bestMove.IsCreature:
		// Pass beats the best found move.
	default:
		action = moveToAction(bestMove)
	}

	if DebugSearchStats {
		elapsed := time.Since(start)
		moveDesc := "pass"
		if usedFallback {
			moveDesc = fmt.Sprintf("fallback:%s", action.Type)
		} else if bestMove != nil && action.Type != interactive.ActionPass {
			moveDesc = fmt.Sprintf("%s:%s", action.Type, bestMove.CardName)
		}
		fmt.Printf("[SEARCH] player=%s nodes=%d depth=%d elapsed=%s budget=%s candidates=%d action=%s\n",
			p.Name(), totalNodes, lastCompletedDepth,
			elapsed.Round(time.Millisecond), s.Config.TimeLimit,
			len(moves), moveDesc)
	}

	return action
}

// searchRoot performs a root-level search across all moves at a given depth
// with the specified alpha-beta window. It returns the best score, best move,
// and the index of the best move in the moves slice.
func (s *SearchStrategy) searchRoot(g *mage.Game, p mage.Player, moves []Move,
	depth, alpha, beta, landsPlayed int, nodes *int, deadline time.Time) (int, *Move, int) {

	depthBestScore := minScore
	var depthBestMove *Move
	depthBestIdx := 0

	for i := range moves {
		m := &moves[i]
		if m.Type == interactive.ActionPass {
			continue
		}

		clone := g.Clone()
		if clone == nil {
			continue
		}
		applyMoveToClone(clone, p.PlayerID(), m, landsPlayed)

		// Late move reduction: reduce depth for moves late in the order
		// that have low heuristic value.
		searchDepth := depth - 1
		if i >= lmrMoveThreshold && depth >= lmrMinDepth && m.heuristic <= 5 {
			searchDepth = max(depth-2, 0)
		}

		score := s.minimax(clone, searchDepth, alpha, beta,
			false, p.PlayerID(), nodes, deadline, 1)

		// Re-search at full depth if LMR found a better score.
		if i >= lmrMoveThreshold && searchDepth < depth-1 && score > depthBestScore {
			score = s.minimax(clone, depth-1, alpha, beta,
				false, p.PlayerID(), nodes, deadline, 1)
		}

		if score > depthBestScore {
			depthBestScore = score
			depthBestMove = m
			depthBestIdx = i
		}

		if *nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
			break
		}
	}

	return depthBestScore, depthBestMove, depthBestIdx
}

// sortByHistory reorders moves by history heuristic score (descending).
func (s *SearchStrategy) sortByHistory(moves []Move) {
	sort.SliceStable(moves, func(i, j int) bool {
		hi := s.history[moveKey(&moves[i])]
		hj := s.history[moveKey(&moves[j])]
		if hi != hj {
			return hi > hj
		}
		return moves[i].heuristic > moves[j].heuristic
	})
}

// ── Attacker / Blocker Search ───────────────────────────────────────────────
//
// Combat decisions delegate to combatsolver. SearchStrategy keeps its own
// outer minimax for priority-action / spell-casting decisions, but combat is
// a constrained subgame that the solver searches exhaustively without sharing
// the outer search's depth/time budget.

func (s *SearchStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	r := combatsolver.SolveAttack(g, p.PlayerID(), combatsolver.Options{
		Profile:  s.solverProfile(),
		Deadline: time.Now().Add(s.Config.TimeLimit),
	})
	if r.DeadlineHit && len(r.Attackers) == 0 {
		return s.Fallback.Attackers(p, g)
	}
	return r.Attackers
}

func (s *SearchStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	r := combatsolver.SolveDefense(g, p.PlayerID(), combatsolver.Options{
		Profile:  s.solverProfile(),
		Deadline: time.Now().Add(s.Config.TimeLimit),
	})
	if r.DeadlineHit && len(r.Blocks) == 0 {
		return s.Fallback.Blockers(p, g)
	}
	return r.Blocks
}

func (s *SearchStrategy) solverProfile() combatsolver.Profile {
	return combatsolver.Profile{
		Weights:        s.Personality.Weights,
		Aggression:     s.Personality.Aggression,
		BlockThreshold: s.Personality.BlockThreshold,
	}
}

// ── Minimax Core ────────────────────────────────────────────────────────────

func (s *SearchStrategy) minimax(g *mage.Game, depth, alpha, beta int,
	maximizing bool, playerID uuid.UUID, nodes *int, deadline time.Time, chainCount int) int {

	*nodes++

	if g.IsGameOver() {
		winner := gameWinner(g, playerID)
		if winner == 1 {
			return maxScore - (s.Config.MaxDepth - depth)
		}
		if winner == -1 {
			return minScore + (s.Config.MaxDepth - depth)
		}
		return 0
	}

	if *nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
		return s.eval(g, playerID)
	}

	if depth <= 0 {
		return s.quiescence(g, alpha, beta, playerID, nodes, deadline, maxQuiescenceDepth)
	}

	var ttHash uint64
	ttActive := s.tt != nil && s.zobrist != nil
	if ttActive {
		ttHash = s.zobrist.SearchKey(g, maximizing, chainCount)
		if score, ok := s.tt.Probe(ttHash, depth, alpha, beta); ok {
			s.TTHits++
			return score
		}
	}
	// alphaOrig/betaOrig capture the window as it stood at function entry
	// (post-TT-probe, pre-move-loop). storeTT uses them to determine the
	// bound flag; comparing against the tightened loop-local alpha/beta
	// would record the wrong bound for entries that failed high or low.
	alphaOrig := alpha
	betaOrig := beta

	var movePlayerID uuid.UUID
	if maximizing {
		movePlayerID = playerID
	} else {
		opp := g.GetOpponent(playerID)
		if opp == nil {
			return s.eval(g, playerID)
		}
		movePlayerID = opp.PlayerID()
	}

	movePlayer := g.GetPlayer(movePlayerID)
	if movePlayer == nil {
		return s.eval(g, playerID)
	}

	// Null-move pruning: if we skip our turn entirely (pass), can we still
	// beat beta? If so, this position is so good we can prune.
	// Only apply when not in a chain and depth is sufficient.
	if depth >= nullMoveReduction+1 && chainCount == 0 && !g.IsGameOver() {
		nullDepth := max(depth-1-nullMoveReduction, 0)
		nullScore := s.minimax(g, nullDepth, alpha, beta, !maximizing, playerID, nodes, deadline, 0)
		// Null-move cutoffs establish a bound on the real score at this node
		// without doing any real work, so they are high-value TT entries.
		if maximizing && nullScore >= beta {
			if ttActive {
				s.storeTT(ttHash, depth, beta, alphaOrig, betaOrig)
			}
			return beta
		}
		if !maximizing && nullScore <= alpha {
			if ttActive {
				s.storeTT(ttHash, depth, alpha, alphaOrig, betaOrig)
			}
			return alpha
		}
	}

	moves := GeneratePriorityMoves(g, movePlayer, g.GetLandsPlayedThisTurn(), g.GetStep().IsMainPhase())

	// Apply personality-driven move ordering adjustments within search.
	s.adjustMoveHeuristics(moves, g, movePlayerID, g.GetStep().IsMainPhase())

	// Apply history heuristic to move ordering (skip pass which is always last).
	if s.history != nil && len(moves) > 2 {
		nonPassEnd := len(moves)
		if moves[len(moves)-1].Type == interactive.ActionPass {
			nonPassEnd = len(moves) - 1
		}
		if nonPassEnd > 1 {
			s.sortByHistory(moves[:nonPassEnd])
		}
	}

	if maximizing {
		best := minScore
		for i := range moves {
			m := &moves[i]
			if m.Type == interactive.ActionPass {
				score := s.minimax(g, depth-1, alpha, beta, false, playerID, nodes, deadline, 0)
				if score > best {
					best = score
				}
				if best > alpha {
					alpha = best
				}
				if alpha >= beta {
					break
				}
				continue
			}

			clone := g.Clone()
			if clone == nil {
				continue
			}
			applyMoveToClone(clone, movePlayerID, m, g.GetLandsPlayedThisTurn())

			// Late move reduction within minimax.
			searchDepth := depth - 1
			if i >= lmrMoveThreshold && depth >= lmrMinDepth && m.heuristic <= 3 {
				searchDepth = max(depth-2, 0)
			}

			var score int
			if chainCount < maxMoveChain {
				score = s.minimax(clone, searchDepth, alpha, beta, true, playerID, nodes, deadline, chainCount+1)
			} else {
				score = s.minimax(clone, searchDepth, alpha, beta, false, playerID, nodes, deadline, 0)
			}

			// Re-search at full depth if LMR result improves alpha.
			if searchDepth < depth-1 && score > alpha {
				if chainCount < maxMoveChain {
					score = s.minimax(clone, depth-1, alpha, beta, true, playerID, nodes, deadline, chainCount+1)
				} else {
					score = s.minimax(clone, depth-1, alpha, beta, false, playerID, nodes, deadline, 0)
				}
			}

			if score > best {
				best = score
			}
			if best > alpha {
				alpha = best
			}
			if alpha >= beta {
				// Record cutoff in history heuristic.
				if s.history != nil {
					s.history[moveKey(m)] += depth * depth
				}
				break
			}
		}
		if ttActive {
			s.storeTT(ttHash, depth, best, alphaOrig, betaOrig)
		}
		return best
	}

	best := maxScore
	for i := range moves {
		m := &moves[i]
		if m.Type == interactive.ActionPass {
			score := s.minimax(g, depth-1, alpha, beta, true, playerID, nodes, deadline, 0)
			if score < best {
				best = score
			}
			if best < beta {
				beta = best
			}
			if alpha >= beta {
				break
			}
			continue
		}

		clone := g.Clone()
		if clone == nil {
			continue
		}
		applyMoveToClone(clone, movePlayerID, m, g.GetLandsPlayedThisTurn())

		// Late move reduction.
		searchDepth := depth - 1
		if i >= lmrMoveThreshold && depth >= lmrMinDepth && m.heuristic <= 3 {
			searchDepth = max(depth-2, 0)
		}

		var score int
		if chainCount < maxMoveChain {
			score = s.minimax(clone, searchDepth, alpha, beta, false, playerID, nodes, deadline, chainCount+1)
		} else {
			score = s.minimax(clone, searchDepth, alpha, beta, true, playerID, nodes, deadline, 0)
		}

		// Re-search at full depth if LMR result improves beta.
		if searchDepth < depth-1 && score < beta {
			if chainCount < maxMoveChain {
				score = s.minimax(clone, depth-1, alpha, beta, false, playerID, nodes, deadline, chainCount+1)
			} else {
				score = s.minimax(clone, depth-1, alpha, beta, true, playerID, nodes, deadline, 0)
			}
		}

		if score < best {
			best = score
		}
		if best < beta {
			beta = best
		}
		if alpha >= beta {
			if s.history != nil {
				s.history[moveKey(m)] += depth * depth
			}
			break
		}
	}
	if ttActive {
		s.storeTT(ttHash, depth, best, alphaOrig, betaOrig)
	}
	return best
}

// storeTT records a minimax result to the transposition table. The flag is
// determined by where `best` sits relative to the original alpha/beta window
// at function entry: below alphaOrig means no move improved alpha (upper
// bound), at/above the original beta means a beta cutoff occurred (lower
// bound), otherwise exact.
func (s *SearchStrategy) storeTT(hash uint64, depth, best, alphaOrig, betaOrig int) {
	flag := TTExact
	if best <= alphaOrig {
		flag = TTUpperBound
	} else if best >= betaOrig {
		flag = TTLowerBound
	}
	s.tt.Store(hash, depth, best, flag)
	s.TTStores++
}

func (s *SearchStrategy) eval(g *mage.Game, playerID uuid.UUID) int {
	return s.Evaluator(g, playerID)
}

// isTactical returns true if the position is non-quiet and warrants
// extended quiescence search rather than a static evaluation.
func isTactical(g *mage.Game, playerID uuid.UUID) bool {
	// Stack is non-empty: spells pending resolution.
	if !g.GetStack().IsEmpty() {
		return true
	}
	// Combat in progress with attackers declared.
	if len(g.CombatGroups()) > 0 {
		return true
	}
	// Opponent at lethal: extending may find the kill.
	opp := g.GetOpponent(playerID)
	if opp != nil {
		totalPower := 0
		for _, perm := range g.FilterBattlefield(mage.IsCreature) {
			if perm.Controller == playerID {
				totalPower += perm.CurrentPower(g)
			}
		}
		if totalPower >= opp.Life() && totalPower > 0 {
			return true
		}
	}
	return false
}

// isTacticalMove returns true if a move is a "tactical" move suitable for
// quiescence search: damage spells, removal, or combat tricks (instants with
// OutcomeDetriment or OutcomeBenefit effect properties).
func isTacticalMove(m *Move) bool {
	return m.IsTactical
}

// quiescence extends the search at leaf nodes when the position is tactical.
// It searches only tactical moves (damage, removal, combat tricks) to avoid
// the horizon effect where the engine stops searching right before a key
// exchange completes.
//
// Standing pat: the side to move can always choose not to act, so the static
// eval serves as a lower bound (for maximizer) or upper bound (for minimizer).
func (s *SearchStrategy) quiescence(g *mage.Game, alpha, beta int,
	playerID uuid.UUID, nodes *int, deadline time.Time, qDepth int) int {

	*nodes++

	if g.IsGameOver() {
		winner := gameWinner(g, playerID)
		if winner == 1 {
			return maxScore
		}
		if winner == -1 {
			return minScore
		}
		return 0
	}

	// If not tactical or out of quiescence depth, return static eval.
	standPat := s.eval(g, playerID)
	if qDepth <= 0 || !isTactical(g, playerID) {
		return standPat
	}

	if *nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
		return standPat
	}

	// Standing pat: the current player can always decline to act.
	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	// Determine whose turn it is: we always search from the maximizer's
	// perspective in quiescence (the player at depth 0 is maximizing).
	movePlayer := g.GetPlayer(playerID)
	if movePlayer == nil {
		return standPat
	}

	moves := GeneratePriorityMoves(g, movePlayer, g.GetLandsPlayedThisTurn(), g.GetStep().IsMainPhase())

	for i := range moves {
		m := &moves[i]
		if m.Type == interactive.ActionPass {
			continue
		}
		// Only consider tactical moves in quiescence.
		if !isTacticalMove(m) {
			continue
		}

		clone := g.Clone()
		if clone == nil {
			continue
		}
		applyMoveToClone(clone, playerID, m, g.GetLandsPlayedThisTurn())

		score := s.quiescence(clone, alpha, beta, playerID, nodes, deadline, qDepth-1)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}

		if *nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
			break
		}
		// Limit quiescence branching: only check the first few tactical moves.
		if i >= 4 {
			break
		}
	}
	return alpha
}

func gameWinner(g *mage.Game, playerID uuid.UUID) int {
	for _, p := range g.AllPlayers() {
		if !p.IsAlive() {
			if p.PlayerID() == playerID {
				return -1
			}
			return 1
		}
	}
	return 0
}

func moveToAction(m *Move) interactive.PriorityAction {
	return interactive.PriorityAction{
		Type:         m.Type,
		CardID:       m.CardID,
		CardName:     m.CardName,
		Targets:      m.Targets,
		PermanentID:  m.PermanentID,
		AbilityIndex: m.AbilityIndex,
		XValue:       m.XValue,
	}
}

// ── Move application on clones using real engine ────────────────────────────

func applyMoveToClone(g *mage.Game, playerID uuid.UUID, m *Move, _ int) {
	switch m.Type {
	case interactive.ActionPlayLand:
		_ = g.PlayLand(playerID, m.CardID)
	case interactive.ActionCastSpell:
		_ = g.CastSpellByID(playerID, m.CardID, m.Targets, m.XValue)
		g.ResolveStack()
	case interactive.ActionActivateAbility:
		_ = g.ActivateAbilityByIndex(playerID, m.PermanentID, m.AbilityIndex, m.Targets)
		g.ResolveStack()
	}
	g.CheckStateBasedActions()
}

// DefaultTTSizeMB is the default transposition-table size in megabytes used
// by NewSearchAI and NewAdaptiveSearchAI. At 16 bytes per entry this fits
// ~262K positions in 4 MB — comfortably in L3 on modern CPUs.
const DefaultTTSizeMB = 4

// NewSearchAI creates an AI player that uses minimax search with a weighted personality.
// The personality's decision weights flow into both the leaf evaluator (Aggression)
// and move ordering (TargetFace, CurvePreference, HoldInstants), so each personality
// produces a distinct play style even under the same search.
func NewSearchAI(name string, config SearchConfig, wp WeightedPersonality) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy: &SearchStrategy{
			Config:      config,
			Evaluator:   eval.NewPersonalityEvaluator(wp.Weights, wp.Aggression),
			Fallback:    NewHeuristicStrategy(wp),
			Personality: wp,
			tt:          NewTranspositionTable(DefaultTTSizeMB),
			zobrist:     DefaultZobrist,
		},
	}
}

// NewAdaptiveSearchAI creates an AI player that uses an AdaptiveStrategy
// where both sub-strategies are SearchStrategy instances: an aggressive
// evaluator when ahead and a defensive evaluator when behind. Each sub-
// strategy gets its own TT so that scores produced by one evaluator can
// never poison probes from the other.
func NewAdaptiveSearchAI(name string, config SearchConfig) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy: &AdaptiveStrategy{
			Aggressive: &SearchStrategy{
				Config:    config,
				Evaluator: eval.NewWeightedEvaluator(AggroWeighted.Weights),
				Fallback:  NewHeuristicStrategy(AggroWeighted),
				tt:        NewTranspositionTable(DefaultTTSizeMB),
				zobrist:   DefaultZobrist,
			},
			Defensive: &SearchStrategy{
				Config:    config,
				Evaluator: eval.NewWeightedEvaluator(ControlWeighted.Weights),
				Fallback:  NewHeuristicStrategy(ControlWeighted),
				tt:        NewTranspositionTable(DefaultTTSizeMB),
				zobrist:   DefaultZobrist,
			},
		},
	}
}
