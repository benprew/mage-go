// Package ai provides computer-controlled players with pluggable strategies
// for the MTG engine. It imports eval/ for position scoring and interactive/
// for the PriorityAction type used by game loops.
package ai

import (
	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
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
// Uses heuristic evaluation based on mode text and game state.
func (ai *AIPlayer) ChooseMode(modes []string, reason string) int {
	return mage.ChooseModeHeuristic(modes, ai.Life(), len(ai.Hand()))
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

// ShouldMulligan evaluates the AI's current hand and returns true if the hand
// should be mulliganed. The handSize parameter is the number of cards currently
// in hand (7 for first mulligan decision, 6 for second, etc.).
func (ai *AIPlayer) ShouldMulligan(handSize int) bool {
	// Never mulligan below 5 cards.
	if handSize <= 5 {
		return false
	}

	hand := ai.Hand()
	lands := 0
	for _, c := range hand {
		if c.HasType(core.TypeLand) {
			lands++
		}
	}
	spells := len(hand) - lands

	if handSize >= 7 {
		// 7-card hand: mulligan on 0-1 lands or 6-7 lands.
		if lands <= 1 || lands >= 6 {
			return true
		}
		// Mulligan if no castable spells: all spells have CMC > lands + 2.
		if spells > 0 {
			hasCastable := false
			for _, c := range hand {
				if !c.HasType(core.TypeLand) && c.ManaCost().CMC() <= lands+2 {
					hasCastable = true
					break
				}
			}
			if !hasCastable {
				return true
			}
		}
		return false
	}

	// 6-card hand: more lenient — mulligan only on 0 or 6 lands.
	if lands == 0 || lands >= 6 {
		return true
	}
	return false
}

// Mulligan shuffles the AI's hand back into the library and draws one fewer card.
func (ai *AIPlayer) Mulligan() {
	hand := ai.Hand()
	handSize := len(hand)
	// Move all cards from hand to library.
	for _, c := range hand {
		ai.AddToLibrary(c)
	}
	ai.SetHand(nil)
	ai.ShuffleLibrary()
	// Draw one fewer card.
	for i := 0; i < handSize-1; i++ {
		ai.DrawCard()
	}
}

// MulliganAI runs the mulligan loop for an AI player. It checks ShouldMulligan
// and calls Mulligan repeatedly until the AI is satisfied or reaches 5 cards.
func MulliganAI(ai *AIPlayer) {
	for len(ai.Hand()) > 5 {
		if !ai.ShouldMulligan(len(ai.Hand())) {
			return
		}
		ai.Mulligan()
	}
}
