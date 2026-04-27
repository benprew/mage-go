package ai

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// AdaptiveStrategy switches between two strategies based on relative life totals.
type AdaptiveStrategy struct {
	Aggressive AIStrategy
	Defensive  AIStrategy
}

func (s *AdaptiveStrategy) active(p mage.Player, g *mage.Game) AIStrategy {
	if eval.DefaultEvaluator(g, p.PlayerID()) >= 0 {
		return s.Aggressive
	}
	return s.Defensive
}

func (s *AdaptiveStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	return s.active(p, g).PriorityAction(p, g, landsPlayed, mainPhase)
}

func (s *AdaptiveStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	return s.active(p, g).Attackers(p, g)
}

func (s *AdaptiveStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	return s.active(p, g).Blockers(p, g)
}

// SequentialStrategy tries each sub-strategy in order and uses the first one
// that produces a non-pass priority action.
type SequentialStrategy struct {
	Strategies []AIStrategy
}

func (s *SequentialStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	for _, start := range s.Strategies {
		if action := start.PriorityAction(p, g, landsPlayed, mainPhase); action.Type != interactive.ActionPass {
			return action
		}
	}
	return interactive.PriorityAction{Type: interactive.ActionPass}
}

func (s *SequentialStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	for _, start := range s.Strategies {
		if atks := start.Attackers(p, g); len(atks) > 0 {
			return atks
		}
	}
	return nil
}

func (s *SequentialStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	for _, start := range s.Strategies {
		if blks := start.Blockers(p, g); len(blks) > 0 {
			return blks
		}
	}
	return nil
}
