// Package search implements a full-turn minimax AI strategy.
package search

import (
	"fmt"
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

	TTHits    uint64
	TTStores  uint64
	LastNodes uint64
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
	res := s.searchAs(g, p.PlayerID(), nil)
	s.recordStats(res)

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
		fmt.Printf("[SEARCH] player=%s nodes=%d tt=%d/%d elapsed=%s action=%s:%s score=%.2f\n",
			p.Name(), res.Nodes, res.TTHits, res.TTStores, time.Since(start).Round(time.Millisecond),
			action.Type, action.CardName, res.Score)
	}

	return action
}

func (s *Strategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	res := s.searchAs(g, p.PlayerID(), nil)
	s.recordStats(res)
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
	res := s.searchAs(searchGame, p.PlayerID(), nil)
	s.recordStats(res)
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
	s.TTHits += uint64(res.TTHits)
	s.TTStores += uint64(res.TTStores)
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
