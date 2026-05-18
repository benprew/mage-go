package search

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

func uuidLess(a, b uuid.UUID) bool { return bytes.Compare(a[:], b[:]) < 0 }

func moveLess(a, b *Move) bool {
	if a.Type != b.Type {
		// On tied scores, prefer doing something over passing. Without this,
		// "cast bolt for lethal now" and "pass, then cast bolt for lethal in
		// postcombat" both score terminalScore and the lower-Type pass wins —
		// the chain then records the lethal move in a later phase and the
		// Strategy adapter, which filters by current step, returns Pass.
		if a.Type == interactive.ActionPass {
			return false
		}
		if b.Type == interactive.ActionPass {
			return true
		}
		return a.Type < b.Type
	}
	if a.CardID != b.CardID {
		return uuidLess(a.CardID, b.CardID)
	}
	if a.PermanentID != b.PermanentID {
		return uuidLess(a.PermanentID, b.PermanentID)
	}
	if a.XValue != b.XValue {
		return a.XValue < b.XValue
	}
	if a.ModeIndex != b.ModeIndex {
		return a.ModeIndex < b.ModeIndex
	}
	if len(a.Targets) != len(b.Targets) {
		return len(a.Targets) < len(b.Targets)
	}
	for i := range a.Targets {
		if a.Targets[i] != b.Targets[i] {
			return uuidLess(a.Targets[i], b.Targets[i])
		}
	}
	return false
}

func attackerSubsetLess(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	for i := range a {
		if a[i] != b[i] {
			return uuidLess(a[i], b[i])
		}
	}
	return false
}

func blockerSubsetLess(a, b []mage.BlockAssignment) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	for i := range a {
		if a[i].AttackerID != b[i].AttackerID {
			return uuidLess(a[i].AttackerID, b[i].AttackerID)
		}
		if a[i].BlockerID != b[i].BlockerID {
			return uuidLess(a[i].BlockerID, b[i].BlockerID)
		}
	}
	return false
}

func attackerSubsetName(g *mage.Game, atk []uuid.UUID) string {
	if len(atk) == 0 {
		return "[]"
	}
	parts := make([]string, len(atk))
	for i, id := range atk {
		if perm := g.FindPermanent(id); perm != nil {
			parts[i] = perm.Name()
		} else {
			parts[i] = id.String()[:8]
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func blockerSubsetName(g *mage.Game, blk []mage.BlockAssignment) string {
	if len(blk) == 0 {
		return "[no blocks]"
	}
	parts := make([]string, len(blk))
	for i, b := range blk {
		bn, an := "?", "?"
		if perm := g.FindPermanent(b.BlockerID); perm != nil {
			bn = perm.Name()
		}
		if perm := g.FindPermanent(b.AttackerID); perm != nil {
			an = perm.Name()
		}
		parts[i] = bn + "->" + an
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func Search(g *mage.Game) Result {
	return SearchAsWithTrace(g, g.ActivePlayerObj().PlayerID(), nil)
}

func SearchWithTrace(g *mage.Game, trace func(string)) Result {
	return SearchAsWithTrace(g, g.ActivePlayerObj().PlayerID(), trace)
}

func SearchAs(g *mage.Game, rootPlayerID uuid.UUID) Result {
	return SearchAsWithTrace(g, rootPlayerID, nil)
}

func SearchAsWithTrace(g *mage.Game, rootPlayerID uuid.UUID, trace func(string)) Result {
	return searchAsWithOptions(g, rootPlayerID, trace, true, DefaultZobrist)
}

func searchAsWithOptions(g *mage.Game, rootPlayerID uuid.UUID, trace func(string), useTT bool, z *ZobristTables) Result {
	if g == nil || g.ActivePlayerObj() == nil {
		return Result{}
	}
	if g.GetStep() == core.DeclareBlockers || rootPlayerID == g.ActivePlayerObj().PlayerID() {
		return searchActiveRoot(g, rootPlayerID, trace, useTT, z)
	}
	return searchInstantResponse(g, rootPlayerID, trace, useTT, z)
}

func searchActiveRoot(g *mage.Game, rootPlayerID uuid.UUID, trace func(string), useTT bool, z *ZobristTables) Result {
	root := g.Clone()
	rootPlayer := root.GetPlayer(rootPlayerID)
	if rootPlayer == nil {
		return Result{}
	}
	var tt map[uint64]simpleTTEntry
	if useTT {
		tt = make(map[uint64]simpleTTEntry)
	}
	s := &searcher{rootPlayer: rootPlayer, trace: trace, tt: tt, zobrist: z}
	chain, score := s.search(root, root.ActivePlayerObj().PlayerID(), false, negInf, posInf)
	return Result{Chain: chain, Score: score, Nodes: s.nodes, MaxDepth: s.maxDepth, TTProbes: s.ttProbes, TTHits: s.ttHits, TTStores: s.ttStores}
}

func searchInstantResponse(g *mage.Game, rootPlayerID uuid.UUID, trace func(string), useTT bool, z *ZobristTables) Result {
	root := g.GetPlayer(rootPlayerID)
	if root == nil {
		return Result{}
	}
	moves := legalPriorityMoves(g, root, true)
	if len(moves) == 0 {
		return Result{}
	}
	step := g.GetStep()

	bestScore := negInf
	var bestMove *Move
	totalNodes := 0
	maxDepth := 0
	ttProbes := 0
	ttHits := 0
	ttStores := 0

	for i := range moves {
		m := &moves[i]
		clone := g.Clone()
		if m.Type != interactive.ActionPass {
			if err := applyMove(clone, rootPlayerID, m); err != nil {
				continue
			}
		}
		var subScore float64
		if clone.IsGameOver() {
			subScore = terminalEval(clone, rootPlayerID)
		} else {
			res := searchActiveRoot(clone, clone.ActivePlayerObj().PlayerID(), nil, useTT, z)
			totalNodes += res.Nodes
			maxDepth = max(maxDepth, res.MaxDepth)
			ttProbes += res.TTProbes
			ttHits += res.TTHits
			ttStores += res.TTStores
			subScore = -res.Score
			if trace != nil {
				name := "pass"
				if m.Type != interactive.ActionPass {
					name = m.CardName
				}
				trace(fmt.Sprintf("    [response] %s score=%.2f", name, subScore))
			}
		}
		if bestMove == nil || subScore > bestScore {
			bestScore = subScore
			bestMove = m
		}
	}

	var chain []ChainStep
	if bestMove != nil {
		mCopy := *bestMove
		chain = []ChainStep{{
			Phase:    step,
			Player:   root.Name(),
			PlayerID: rootPlayerID,
			StateKey: turnPlanDecisionKey(z, g, rootPlayerID, rootPlayerID, turnPlanPriority),
			Move:     &mCopy,
		}}
	}
	return Result{Chain: chain, Score: bestScore, Nodes: totalNodes, MaxDepth: maxDepth, TTProbes: ttProbes, TTHits: ttHits, TTStores: ttStores}
}

type searcher struct {
	rootPlayer mage.Player
	nodes      int
	trace      func(string)
	tt         map[uint64]simpleTTEntry
	zobrist    *ZobristTables
	depth      int
	maxDepth   int
	ttProbes   int
	ttHits     int
	ttStores   int
}

func (s *searcher) search(g *mage.Game, priorityID uuid.UUID, prevPass bool, alpha, beta float64) (chain []ChainStep, score float64) {
	s.nodes++
	s.depth++
	if s.depth > s.maxDepth {
		s.maxDepth = s.depth
	}
	defer func() {
		s.depth--
	}()
	key := s.ttKey(g, priorityID, prevPass)
	if entry, ok := s.probeTT(key, alpha, beta); ok {
		s.ttHits++
		return nil, entry.score
	}
	alphaOrig, betaOrig := alpha, beta
	defer func() {
		s.storeTT(key, chain, score, alphaOrig, betaOrig)
	}()

	if g.IsGameOver() {
		return nil, terminalEval(g, s.rootPlayer.PlayerID())
	}

	step := g.GetStep()
	switch step {
	case core.Cleanup:
		clone := g.Clone()
		clone.SetOnPriority(passOnly)
		clone.RunStepWithPriority(core.Cleanup)
		if clone.IsGameOver() {
			return nil, terminalEval(clone, s.rootPlayer.PlayerID())
		}
		score := evaluate(clone, s.rootPlayer.PlayerID())
		if s.trace != nil {
			me := clone.GetPlayer(s.rootPlayer.PlayerID())
			opp := clone.GetOpponent(s.rootPlayer.PlayerID())
			myBoard, oppBoard := boardPower(clone, s.rootPlayer.PlayerID())
			s.trace(fmt.Sprintf("        [eval] life=%d/%d board=%.1f/%.1f score=%.2f",
				me.Life(), opp.Life(), myBoard, oppBoard, score))
		}
		return nil, score
	case core.DeclareAttackers:
		return s.branchAttackers(g, alpha, beta)
	case core.DeclareBlockers:
		return s.branchBlockers(g, alpha, beta)
	}

	if step.IsMainPhase() {
		return s.priorityRound(g, priorityID, prevPass, alpha, beta)
	}

	clone := g.Clone()
	clone.SetOnPriority(passOnly)
	clone.RunStepWithPriority(step)
	if clone.IsGameOver() {
		return nil, terminalEval(clone, s.rootPlayer.PlayerID())
	}
	clone.SetStep(nextStep(step))
	return s.search(clone, clone.ActivePlayerObj().PlayerID(), false, alpha, beta)
}

func (s *searcher) priorityRound(g *mage.Game, priorityID uuid.UUID, prevPass bool, alpha, beta float64) ([]ChainStep, float64) {
	step := g.GetStep()
	actor := g.GetPlayer(priorityID)
	if actor == nil {
		return nil, evaluate(g, s.rootPlayer.PlayerID())
	}
	moves := legalPriorityMoves(g, actor, priorityID == s.rootPlayer.PlayerID())

	isMax := priorityID == s.rootPlayer.PlayerID()
	bestScore := negInf
	if !isMax {
		bestScore = posInf
	}
	var bestChain []ChainStep
	var bestMove *Move

	for i := range moves {
		m := &moves[i]
		var subChain []ChainStep
		var subScore float64

		if m.Type == interactive.ActionPass {
			if prevPass {
				origStep := g.GetStep()
				g.SetStep(nextStep(step))
				subChain, subScore = s.search(g, g.ActivePlayerObj().PlayerID(), false, alpha, beta)
				g.SetStep(origStep)
			} else {
				other := otherPlayerID(g, priorityID)
				subChain, subScore = s.search(g, other, true, alpha, beta)
			}
		} else {
			clone := g.Clone()
			if err := applyMove(clone, priorityID, m); err != nil {
				continue
			}
			subChain, subScore = s.search(clone, priorityID, false, alpha, beta)
		}

		better := false
		if isMax {
			if subScore > bestScore || (subScore == bestScore && (bestMove == nil || moveLess(m, bestMove))) {
				better = true
			}
			if subScore > alpha {
				alpha = subScore
			}
		} else {
			if subScore < bestScore || (subScore == bestScore && (bestMove == nil || moveLess(m, bestMove))) {
				better = true
			}
			if subScore < beta {
				beta = subScore
			}
		}
		if better {
			mCopy := *m
			record := ChainStep{
				Phase:    step,
				Player:   actor.Name(),
				PlayerID: priorityID,
				StateKey: s.planDecisionKey(g, priorityID, turnPlanPriority),
				Move:     &mCopy,
			}
			bestScore = subScore
			bestChain = append([]ChainStep{record}, subChain...)
			bestMove = m
		}
		if alpha >= beta {
			break
		}
	}

	return bestChain, bestScore
}

func (s *searcher) branchAttackers(g *mage.Game, alpha, beta float64) ([]ChainStep, float64) {
	activeIdx := g.ActivePlayerIndex()
	activePlayer := g.PlayerAt(activeIdx)
	activePlayerID := activePlayer.PlayerID()
	subsets := attackerSubsets(g, activePlayerID)

	if len(subsets) == 0 {
		clone := g.Clone()
		clone.SetOnPriority(passOnly)
		clone.RunStepWithPriority(core.DeclareAttackers)
		if clone.IsGameOver() {
			return nil, terminalEval(clone, s.rootPlayer.PlayerID())
		}
		clone.SetStep(nextStep(core.DeclareAttackers))
		return s.search(clone, clone.ActivePlayerObj().PlayerID(), false, alpha, beta)
	}

	isMax := activePlayerID == s.rootPlayer.PlayerID()
	bestScore := negInf
	if !isMax {
		bestScore = posInf
	}
	var bestChain []ChainStep
	var bestSubset []uuid.UUID
	bestSet := false

	for _, atk := range subsets {
		clone := g.Clone()
		clone.SetOnPriority(passOnly)
		installAttackerOverride(clone, activeIdx, atk)
		clone.RunStepWithPriority(core.DeclareAttackers)

		var subChain []ChainStep
		var subScore float64
		if clone.IsGameOver() {
			subScore = terminalEval(clone, s.rootPlayer.PlayerID())
		} else {
			clone.SetStep(nextStep(core.DeclareAttackers))
			subChain, subScore = s.search(clone, clone.ActivePlayerObj().PlayerID(), false, alpha, beta)
		}

		if s.trace != nil {
			s.trace(fmt.Sprintf("    attackers=%s score=%.2f", attackerSubsetName(g, atk), subScore))
		}

		better := false
		if isMax {
			if subScore > bestScore || (subScore == bestScore && (!bestSet || attackerSubsetLess(atk, bestSubset))) {
				better = true
			}
			if subScore > alpha {
				alpha = subScore
			}
		} else {
			if subScore < bestScore || (subScore == bestScore && (!bestSet || attackerSubsetLess(atk, bestSubset))) {
				better = true
			}
			if subScore < beta {
				beta = subScore
			}
		}
		if better {
			attackers := append([]uuid.UUID{}, atk...)
			record := ChainStep{
				Phase:     core.DeclareAttackers,
				Player:    activePlayer.Name(),
				PlayerID:  activePlayerID,
				StateKey:  s.planDecisionKey(g, activePlayerID, turnPlanAttackers),
				Attackers: attackers,
			}
			bestScore = subScore
			bestChain = append([]ChainStep{record}, subChain...)
			bestSubset = atk
			bestSet = true
		}
		if alpha >= beta {
			break
		}
	}

	return bestChain, bestScore
}

func (s *searcher) branchBlockers(g *mage.Game, alpha, beta float64) ([]ChainStep, float64) {
	defenderIdx := 1 - g.ActivePlayerIndex()
	defender := g.PlayerAt(defenderIdx)
	defenderID := defender.PlayerID()
	subsets := blockerSubsets(g, defenderID)

	if len(subsets) == 0 {
		clone := g.Clone()
		clone.SetOnPriority(passOnly)
		clone.RunStepWithPriority(core.DeclareBlockers)
		if clone.IsGameOver() {
			return nil, terminalEval(clone, s.rootPlayer.PlayerID())
		}
		clone.SetStep(nextStep(core.DeclareBlockers))
		return s.search(clone, clone.ActivePlayerObj().PlayerID(), false, alpha, beta)
	}

	isMax := defenderID == s.rootPlayer.PlayerID()
	bestScore := negInf
	if !isMax {
		bestScore = posInf
	}
	var bestChain []ChainStep
	var bestSubset []mage.BlockAssignment
	bestSet := false

	for _, blk := range subsets {
		clone := g.Clone()
		clone.SetOnPriority(passOnly)
		installBlockerOverride(clone, defenderIdx, blk)
		clone.RunStepWithPriority(core.DeclareBlockers)

		var subChain []ChainStep
		var subScore float64
		if clone.IsGameOver() {
			subScore = terminalEval(clone, s.rootPlayer.PlayerID())
		} else {
			clone.SetStep(nextStep(core.DeclareBlockers))
			subChain, subScore = s.search(clone, clone.ActivePlayerObj().PlayerID(), false, alpha, beta)
		}

		if s.trace != nil {
			s.trace(fmt.Sprintf("      blockers=%s score=%.2f", blockerSubsetName(g, blk), subScore))
		}

		better := false
		if isMax {
			if subScore > bestScore || (subScore == bestScore && (!bestSet || blockerSubsetLess(blk, bestSubset))) {
				better = true
			}
			if subScore > alpha {
				alpha = subScore
			}
		} else {
			if subScore < bestScore || (subScore == bestScore && (!bestSet || blockerSubsetLess(blk, bestSubset))) {
				better = true
			}
			if subScore < beta {
				beta = subScore
			}
		}
		if better {
			blocks := append([]mage.BlockAssignment{}, blk...)
			record := ChainStep{
				Phase:    core.DeclareBlockers,
				Player:   defender.Name(),
				PlayerID: defenderID,
				StateKey: s.planDecisionKey(g, defenderID, turnPlanBlockers),
				Blocks:   blocks,
			}
			bestScore = subScore
			bestChain = append([]ChainStep{record}, subChain...)
			bestSubset = blk
			bestSet = true
		}
		if alpha >= beta {
			break
		}
	}

	return bestChain, bestScore
}

func nextStep(s core.PhaseStep) core.PhaseStep {
	all := core.AllSteps()
	for i, step := range all {
		if step == s && i+1 < len(all) {
			return all[i+1]
		}
	}
	return core.Cleanup
}

func otherPlayerID(g *mage.Game, id uuid.UUID) uuid.UUID {
	for i := range 2 {
		p := g.PlayerAt(i)
		if p.PlayerID() != id {
			return p.PlayerID()
		}
	}
	return uuid.UUID{}
}
