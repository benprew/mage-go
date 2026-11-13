package heuristic

import (
	"math"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

func isSimulatedPermanent(card *mage.Card) bool {
	return card.HasType(core.TypeCreature) || card.HasType(core.TypeArtifact) || card.HasType(core.TypeEnchantment)
}

func simulatePermanentCast(g *mage.Game, playerID uuid.UUID, card *mage.Card, targets []uuid.UUID, x int) *mage.Game {
	if len(g.StackObjects()) != 0 || !isSimulatedPermanent(card) {
		return nil
	}
	return simulateSpellCast(g, playerID, card, targets, x)
}

func (s *Strategy) permanentCastValue(g *mage.Game, playerID uuid.UUID, card *mage.Card, targets []uuid.UUID, x int) (int, bool) {
	branch := simulatePermanentCast(g, playerID, card, targets, x)
	if branch == nil {
		return 0, false
	}
	cardValue := s.solverProfile().Card
	value := func(position *mage.Game) float64 { return s.castPositionValue(position, playerID) }
	// Other spell scores exclude the card spent to cast them.
	best := value(branch) + cardValue
	// A second cast can use an effect from the first permanent.
	if !branch.IsGameOver() {
		player := branch.GetPlayer(playerID)
		for _, next := range branch.GetCastableSpells(playerID) {
			if !isSimulatedPermanent(next) || !aiHintAllowsTiming(cardAIHint(next), branch, branch.GetStep().IsMainPhase()) {
				continue
			}
			nextTargets := s.autoSelectTargets(player, branch, next)
			if requiresTargets(next) && len(nextTargets) == 0 {
				continue
			}
			if held := instantToHold(player, branch, s.weights()); held != nil && (next.ManaCost().HasX || !canCastWhileReserving(branch, playerID, next, held)) {
				continue
			}
			x := s.bestXValue(branch, playerID, next, nextTargets)
			if followup := simulatePermanentCast(branch, playerID, next, nextTargets, x); followup != nil {
				best = max(best, value(followup)+2*cardValue)
			}
		}
	}
	return roundCastValue(best - value(g)), true
}

func roundCastValue(gain float64) int {
	if gain > 0 {
		return int(math.Ceil(gain))
	}
	return int(math.Floor(gain))
}

func simulateSpellCast(g *mage.Game, playerID uuid.UUID, card *mage.Card, targets []uuid.UUID, x int) *mage.Game {
	clone := g.Clone()
	if err := clone.CastSpellByID(playerID, card.ID(), targets, x); err != nil {
		return nil
	}
	clone.ResolveStack()
	clone.CheckStateBasedActions()
	return clone
}

func (s *Strategy) castPositionValue(position *mage.Game, playerID uuid.UUID) float64 {
	weights := s.solverProfile().Weights
	// Mana reservation and curve scoring account for available mana.
	weights.Tempo = 0
	if position.IsGameOver() {
		switch position.Winner() {
		case position.GetPlayer(playerID).Name():
			return 100000
		case position.GetOpponent(playerID).Name():
			return -100000
		default:
			return 0
		}
	}
	score := eval.WeightedPositionValue(position, playerID, weights)
	for _, permanent := range position.AllBattlefield() {
		if !permanent.HasType(core.TypeCreature) {
			continue
		}
		bonus := float64(unansweredEvasionValue(position, permanent, permanent.CurrentPower(position))) * weights.Board / 2
		if permanent.ControllerID() == playerID {
			score += bonus
		} else {
			score -= bonus
		}
	}
	return score
}
