package heuristic

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// Adaptive switches between two strategies based on relative life totals.
type Adaptive struct {
	Aggressive ai.AIStrategy
	Defensive  ai.AIStrategy
}

// NewAdaptive returns an Adaptive that plays Aggro when ahead and Control when behind.
func NewAdaptive() *Adaptive {
	return &Adaptive{
		Aggressive: New(ai.AggroWeighted),
		Defensive:  New(ai.ControlWeighted),
	}
}

func (s *Adaptive) active(p mage.Player, g *mage.Game) ai.AIStrategy {
	if eval.DefaultEvaluator(g, p.PlayerID()) >= 0 {
		return s.Aggressive
	}
	return s.Defensive
}

func (s *Adaptive) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	return s.active(p, g).PriorityAction(p, g, landsPlayed, mainPhase)
}

func (s *Adaptive) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	return s.active(p, g).Attackers(p, g)
}

func (s *Adaptive) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	return s.active(p, g).Blockers(p, g)
}

// Sequential tries each sub-strategy in order and uses the first one that
// produces a non-pass priority action.
type Sequential struct {
	Strategies []ai.AIStrategy
}

func (s *Sequential) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	for _, start := range s.Strategies {
		if action := start.PriorityAction(p, g, landsPlayed, mainPhase); action.Type != interactive.ActionPass {
			return action
		}
	}
	return interactive.PriorityAction{Type: interactive.ActionPass}
}

func (s *Sequential) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	for _, start := range s.Strategies {
		if atks := start.Attackers(p, g); len(atks) > 0 {
			return atks
		}
	}
	return nil
}

func (s *Sequential) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	for _, start := range s.Strategies {
		if blks := start.Blockers(p, g); len(blks) > 0 {
			return blks
		}
	}
	return nil
}
