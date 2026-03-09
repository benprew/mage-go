package mage

import (
	"strings"

	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
)

// SearchPlayer wraps a BasePlayer for use in cloned game states during AI search.
// It implements the Player interface with reasonable default choices for all
// interactive methods, allowing the game engine to run without human input.
type SearchPlayer struct {
	*BasePlayer
}

// NewSearchPlayer creates a SearchPlayer wrapping the given BasePlayer.
// The BasePlayer's UUID is preserved (critical for search — moves reference UUIDs).
func NewSearchPlayer(bp *BasePlayer) *SearchPlayer {
	return &SearchPlayer{BasePlayer: bp}
}

// ChooseTargets returns the first min valid targets (first-available default).
func (sp *SearchPlayer) ChooseTargets(possible []uuid.UUID, min, max int, g *Game) []uuid.UUID {
	if len(possible) >= min {
		return possible[:min]
	}
	return nil
}

// DeclareAttackers returns empty — search player does not attack by default.
func (sp *SearchPlayer) DeclareAttackers(g *Game) []uuid.UUID {
	return nil
}

// DeclareBlockers returns empty — search player does not block by default.
func (sp *SearchPlayer) DeclareBlockers(g *Game) []BlockAssignment {
	return nil
}

// ChooseMayAbility accepts if the description contains "draw", "damage", or "destroy".
// Declines unknown optional abilities.
func (sp *SearchPlayer) ChooseMayAbility(description string) bool {
	lower := strings.ToLower(description)
	return strings.Contains(lower, "draw") ||
		strings.Contains(lower, "damage") ||
		strings.Contains(lower, "destroy")
}

// ChooseMode uses the shared mode heuristic based on life and hand size.
func (sp *SearchPlayer) ChooseMode(modes []string, reason string) int {
	return ChooseModeHeuristic(modes, sp.Life(), len(sp.Hand()))
}

// ChoosePermanent picks based on context:
//   - "sacrifice": pick lowest-value permanent (by CMC)
//   - "destroy": pick highest-value permanent (by CMC)
//   - default: first candidate
func (sp *SearchPlayer) ChoosePermanent(candidates []*Permanent, reason string, g GameReader) *Permanent {
	if len(candidates) == 0 {
		return nil
	}
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "sacrifice") {
		// Pick lowest-value: cheapest CMC
		best := candidates[0]
		bestCMC := best.Card.ManaCost().CMC()
		for _, c := range candidates[1:] {
			cmc := c.Card.ManaCost().CMC()
			if cmc < bestCMC {
				best = c
				bestCMC = cmc
			}
		}
		return best
	}
	if strings.Contains(lower, "destroy") {
		// Pick highest-value: most expensive CMC
		best := candidates[0]
		bestCMC := best.Card.ManaCost().CMC()
		for _, c := range candidates[1:] {
			cmc := c.Card.ManaCost().CMC()
			if cmc > bestCMC {
				best = c
				bestCMC = cmc
			}
		}
		return best
	}
	return candidates[0]
}

// ChooseCardsFromHand returns the cheapest cards (lowest CMC first).
func (sp *SearchPlayer) ChooseCardsFromHand(amount int, reason string, g GameReader) []Card {
	hand := sp.Hand()
	if amount > len(hand) {
		amount = len(hand)
	}
	if amount == 0 {
		return nil
	}
	// Sort by CMC to discard cheapest.
	type cardCMC struct {
		card Card
		cmc  int
	}
	sorted := make([]cardCMC, len(hand))
	for i, c := range hand {
		sorted[i] = cardCMC{card: c, cmc: c.ManaCost().CMC()}
	}
	// Simple selection sort for cheapest (hand is small).
	for i := 0; i < amount && i < len(sorted); i++ {
		minIdx := i
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].cmc < sorted[minIdx].cmc {
				minIdx = j
			}
		}
		sorted[i], sorted[minIdx] = sorted[minIdx], sorted[i]
	}
	result := make([]Card, amount)
	for i := 0; i < amount; i++ {
		result[i] = sorted[i].card
	}
	return result
}

// ChooseManaColor returns the first available color in the pool,
// falling back to White if the pool is empty.
func (sp *SearchPlayer) ChooseManaColor(reason string) Color {
	pool := sp.ManaPool()
	for _, c := range []Color{White, Blue, Black, Red, Green, Colorless} {
		if pool.Count(c) > 0 {
			return c
		}
	}
	return White
}

// ChooseCardFromLibrary returns the first candidate.
func (sp *SearchPlayer) ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}

// ChooseNumber picks based on context:
//   - "damage" context: pick max (deal maximum damage)
//   - "discard" context: pick min (discard as few as possible)
//   - default: pick max (greedy)
func (sp *SearchPlayer) ChooseNumber(min, max int, reason string) int {
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "discard") {
		return min
	}
	return max
}

// Compile-time check that SearchPlayer satisfies Player.
var _ Player = (*SearchPlayer)(nil)
