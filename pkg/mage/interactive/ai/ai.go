// Package ai provides computer-controlled players with pluggable strategies
// for the MTG engine. It imports eval/ for position scoring and interactive/
// for the PriorityAction type used by game loops.
package ai

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/interactive"
)

// AIStrategy is the decision-making interface for computer-controlled players.
type AIStrategy interface {
	PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction
	Attackers(p mage.Player, g *mage.Game) []uuid.UUID
	Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment
}

// AIPlayer is a computer-controlled player with a pluggable AIStrategy.
type AIPlayer struct {
	*mage.BasePlayer
	strategy AIStrategy
}

// NewAIPlayer creates an AI player with the default Midrange personality.
func NewAIPlayer(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(MidrangeWeighted),
	}
}

// NewAggroAI creates an AI player with the Aggro personality.
func NewAggroAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(AggroWeighted),
	}
}

// NewControlAI creates an AI player with the Control personality.
func NewControlAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(ControlWeighted),
	}
}

// NewTempoAI creates an AI player with the Tempo personality.
func NewTempoAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(TempoWeighted),
	}
}

// NewWeightedAI creates an AI player with a custom WeightedPersonality.
func NewWeightedAI(name string, w WeightedPersonality) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(w),
	}
}

// NewBurnAI creates an AI player focused on dealing direct damage.
func NewBurnAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(BurnWeighted),
	}
}

// NewAdaptiveAI creates an AI that plays aggressively when ahead and switches
// to a controlling game plan when behind on life.
func NewAdaptiveAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy: &AdaptiveStrategy{
			Aggressive: NewHeuristicStrategy(AggroWeighted),
			Defensive:  NewHeuristicStrategy(ControlWeighted),
		},
	}
}

// ChooseMode implements the Player interface for AI mode selection.
func (ai *AIPlayer) ChooseMode(modes []string, reason string) int {
	switch reason {
	case "Healing Salve":
		if ai.Life() <= 10 {
			return 0
		}
		return 1
	}
	return 0
}

// GetPriorityAction decides what the AI should do when it has priority.
func (ai *AIPlayer) GetPriorityAction(g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	return ai.strategy.PriorityAction(ai.BasePlayer, g, landsPlayed, mainPhase)
}

// AIAttackers returns the IDs of creatures the AI wants to attack with.
func (ai *AIPlayer) AIAttackers(g *mage.Game) []uuid.UUID {
	return ai.strategy.Attackers(ai.BasePlayer, g)
}

// AIBlockers returns blocker->attacker assignments.
func (ai *AIPlayer) AIBlockers(g *mage.Game) []mage.BlockAssignment {
	return ai.strategy.Blockers(ai.BasePlayer, g)
}

// DeclareAttackers implements the Player interface for AI.
func (ai *AIPlayer) DeclareAttackers(g *mage.Game) []uuid.UUID {
	return ai.AIAttackers(g)
}

// DeclareBlockers implements the Player interface for AI.
func (ai *AIPlayer) DeclareBlockers(g *mage.Game) []mage.BlockAssignment {
	return ai.AIBlockers(g)
}
