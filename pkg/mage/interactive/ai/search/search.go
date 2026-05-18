// Package search implements a full-turn minimax AI strategy.
package search

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// DebugStats enables one-line per-decision logging of search telemetry.
var DebugStats bool

// Config controls search limits. The unified search currently uses these as
// strategy metadata; the full-turn search terminates at cleanup.
type Config struct {
	MaxDepth  int
	MaxNodes  int
	TimeLimit time.Duration
}

// DefaultConfig returns the default search configuration.
func DefaultConfig() Config {
	return Config{
		MaxDepth:  10,
		MaxNodes:  20000,
		TimeLimit: 1500 * time.Millisecond,
	}
}

// DefaultTTSizeMB is retained for callers that construct TT-enabled strategies.
const DefaultTTSizeMB = 4

// Strategy implements ai.AIStrategy using one unified full-turn minimax for
// priority actions, attacker declarations, and blocker assignments.
type Strategy struct {
	Config      Config
	Evaluator   eval.StateEvaluator
	Fallback    ai.AIStrategy
	Personality ai.WeightedPersonality

	tt      *TranspositionTable
	zobrist *ZobristTables

	turnPlan *turnPlan

	TTHits    uint64
	TTProbes  uint64
	TTStores  uint64
	LastNodes uint64
	LastDepth uint64
}

// New creates a search-based Strategy.
func New(cfg Config, wp ai.WeightedPersonality) *Strategy {
	return &Strategy{
		Config:      cfg,
		Evaluator:   eval.NewPersonalityEvaluator(wp.Weights, wp.Aggression),
		Personality: wp,
		tt:          NewTranspositionTable(DefaultTTSizeMB),
		zobrist:     DefaultZobrist,
	}
}

// NewAdaptive returns a search-backed strategy that switches between an
// aggressive and a defensive search based on relative life totals.
func NewAdaptive(cfg Config) ai.AIStrategy {
	return &adaptiveStrategy{
		Aggressive: &Strategy{
			Config:      cfg,
			Evaluator:   eval.NewWeightedEvaluator(ai.AggroWeighted.Weights),
			Personality: ai.AggroWeighted,
			tt:          NewTranspositionTable(DefaultTTSizeMB),
			zobrist:     DefaultZobrist,
		},
		Defensive: &Strategy{
			Config:      cfg,
			Evaluator:   eval.NewWeightedEvaluator(ai.ControlWeighted.Weights),
			Personality: ai.ControlWeighted,
			tt:          NewTranspositionTable(DefaultTTSizeMB),
			zobrist:     DefaultZobrist,
		},
	}
}

type adaptiveStrategy struct {
	Aggressive ai.AIStrategy
	Defensive  ai.AIStrategy
}

func (s *adaptiveStrategy) active(p mage.Player, g *mage.Game) ai.AIStrategy {
	if eval.DefaultEvaluator(g, p.PlayerID()) >= 0 {
		return s.Aggressive
	}
	return s.Defensive
}

func (s *adaptiveStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	return s.active(p, g).PriorityAction(p, g, landsPlayed, mainPhase)
}

func (s *adaptiveStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	return s.active(p, g).Attackers(p, g)
}

func (s *adaptiveStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	return s.active(p, g).Blockers(p, g)
}

func (s *Strategy) PriorityAction(p mage.Player, g *mage.Game, _ int, _ bool) interactive.PriorityAction {
	start := time.Now()
	if skipEarlyStepSearch(g) {
		s.LastNodes = 0
		s.LastDepth = 0
		if DebugStats {
			fmt.Fprintf(os.Stderr, "[SEARCH] player=%s nodes=0 depth=0 tt=0/0 hit=0.0%% elapsed=%s action=%s:%s score=skip\n",
				p.Name(), time.Since(start).Round(time.Millisecond), interactive.ActionPass, "")
		}
		return interactive.PriorityAction{Type: interactive.ActionPass}
	}
	if action, ok := s.replayPriorityAction(p, g); ok {
		s.LastNodes = 0
		s.LastDepth = 0
		if DebugStats {
			fmt.Fprintf(os.Stderr, "[SEARCH] player=%s nodes=0 depth=0 tt=0/0 hit=0.0%% elapsed=%s action=%s:%s score=plan\n",
				p.Name(), time.Since(start).Round(time.Millisecond), action.Type, action.CardName)
		}
		return action
	}

	res := s.searchAs(g, p.PlayerID(), nil)
	s.recordStats(res)
	s.storeTurnPlan(g, p.PlayerID(), res)

	action := interactive.PriorityAction{Type: interactive.ActionPass}
	step := g.GetStep()
	for _, chainStep := range res.Chain {
		if chainStep.Phase == step && chainStep.Move != nil && chainStep.Player == p.Name() {
			action = moveToAction(chainStep.Move)
			break
		}
	}

	if action.Type == interactive.ActionPass && s.Fallback != nil && len(GeneratePriorityMoves(g, p, g.GetLandsPlayedThisTurn(), g.GetStep().IsMainPhase())) > 1 {
		// If the PV contains no current-window priority action, pass is the
		// searched choice. Keep fallback only for malformed/null search output.
		if res.Nodes == 0 {
			action = s.Fallback.PriorityAction(p, g, g.GetLandsPlayedThisTurn(), g.GetStep().IsMainPhase())
		}
	}

	if DebugStats {
		fmt.Fprintf(os.Stderr, "[SEARCH] player=%s nodes=%d depth=%d tt=%d/%d hit=%.1f%% elapsed=%s action=%s:%s score=%.2f\n",
			p.Name(), res.Nodes, res.MaxDepth, res.TTHits, res.TTProbes, ttHitRate(res), time.Since(start).Round(time.Millisecond),
			action.Type, action.CardName, res.Score)
	}

	return action
}

func skipEarlyStepSearch(g *mage.Game) bool {
	if g == nil || g.StackSize() != 0 {
		return false
	}
	step := g.GetStep()
	return step == core.Upkeep || step == core.Draw
}

func (s *Strategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	if attackers, ok := s.replayAttackers(p, g); ok {
		return attackers
	}

	res := s.searchAs(g, p.PlayerID(), nil)
	s.recordStats(res)
	s.storeTurnPlan(g, p.PlayerID(), res)
	for _, step := range res.Chain {
		if step.Attackers != nil {
			return step.Attackers
		}
	}
	if s.Fallback != nil && res.Nodes == 0 {
		return s.Fallback.Attackers(p, g)
	}
	return nil
}

func (s *Strategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	searchGame := g
	if g.GetStep() != core.DeclareBlockers && len(g.CombatGroups()) > 0 {
		searchGame = g.Clone()
		searchGame.SetStep(core.DeclareBlockers)
	}
	if blocks, ok := s.replayBlockers(p, searchGame); ok {
		return blocks
	}

	res := s.searchAs(searchGame, p.PlayerID(), nil)
	s.recordStats(res)
	s.storeTurnPlan(searchGame, p.PlayerID(), res)
	for _, step := range res.Chain {
		if step.Blocks != nil {
			return step.Blocks
		}
	}
	if s.Fallback != nil && res.Nodes == 0 {
		return s.Fallback.Blockers(p, g)
	}
	return nil
}

func (s *Strategy) searchAs(g *mage.Game, rootPlayerID uuid.UUID, trace func(string)) Result {
	useTT := s != nil && s.tt != nil && s.zobrist != nil
	z := DefaultZobrist
	if s != nil && s.zobrist != nil {
		z = s.zobrist
	}
	return searchAsWithOptions(g, rootPlayerID, trace, useTT, z)
}

func (s *Strategy) recordStats(res Result) {
	s.LastNodes = uint64(res.Nodes)
	s.LastDepth = uint64(res.MaxDepth)
	s.TTProbes += uint64(res.TTProbes)
	s.TTHits += uint64(res.TTHits)
	s.TTStores += uint64(res.TTStores)
}

func ttHitRate(res Result) float64 {
	if res.TTProbes == 0 {
		return 0
	}
	return 100 * float64(res.TTHits) / float64(res.TTProbes)
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
