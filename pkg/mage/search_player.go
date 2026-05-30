package mage

import (
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
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

// permanentValue scores a permanent for targeting priority. Higher = more valuable target.
// Uses current P/T for creatures, CMC for everything else.
func permanentValue(p *Permanent, g GameReader) int {
	if p.HasType(TypeCreature) {
		return p.CurrentPower(g) + p.CurrentToughness(g)
	}
	return p.Card.ManaCost().CMC()
}

// ChooseTargets selects targets with context-aware heuristics:
//   - Prefers opponent's permanents over own (most targeted spells are removal)
//   - Among opponent permanents, prefers highest-value creatures
//   - If targets include player IDs, prefers the opponent
func (sp *SearchPlayer) ChooseTargets(possible []uuid.UUID, min, max int, g *Game) []uuid.UUID {
	if len(possible) < min {
		return nil
	}

	// If no game context, fall back to first-available.
	if g == nil {
		count := min
		if count > len(possible) {
			count = len(possible)
		}
		return possible[:count]
	}

	myID := sp.PlayerID()

	// Partition targets into categories.
	type scoredTarget struct {
		id    uuid.UUID
		score int // higher = prefer as target
	}
	scored := make([]scoredTarget, 0, len(possible))

	for _, id := range possible {
		if perm := g.FindPermanent(id); perm != nil {
			val := permanentValue(perm, g)
			if perm.Controller != myID {
				// Opponent's permanent: high priority (removal heuristic).
				scored = append(scored, scoredTarget{id, 1000 + val})
			} else {
				// Own permanent: low priority as a target.
				scored = append(scored, scoredTarget{id, val})
			}
		} else if player := g.GetPlayer(id); player != nil {
			if player.PlayerID() != myID {
				// Opponent player: prefer targeting them.
				scored = append(scored, scoredTarget{id, 2000})
			} else {
				// Self: avoid targeting.
				scored = append(scored, scoredTarget{id, 0})
			}
		} else {
			// Unknown target (e.g., stack object): neutral priority.
			scored = append(scored, scoredTarget{id, 500})
		}
	}

	// Selection sort descending by score (possible list is small).
	for i := 0; i < len(scored)-1; i++ {
		maxIdx := i
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[maxIdx].score {
				maxIdx = j
			}
		}
		scored[i], scored[maxIdx] = scored[maxIdx], scored[i]
	}

	count := min
	if count > len(scored) {
		count = len(scored)
	}
	result := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		result[i] = scored[i].id
	}
	return result
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
//   - "sacrifice": pick lowest-value permanent (by P/T for creatures, CMC otherwise)
//   - "destroy": pick highest-value permanent (by P/T for creatures, CMC otherwise)
//   - default: first candidate
func (sp *SearchPlayer) ChoosePermanent(candidates []*Permanent, reason string, g GameReader) *Permanent {
	if len(candidates) == 0 {
		return nil
	}
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "sacrifice") {
		// Pick lowest-value permanent.
		best := candidates[0]
		bestVal := permValueForChoice(best, g)
		for _, c := range candidates[1:] {
			val := permValueForChoice(c, g)
			if val < bestVal {
				best = c
				bestVal = val
			}
		}
		return best
	}
	if strings.Contains(lower, "destroy") {
		// Pick highest-value permanent.
		best := candidates[0]
		bestVal := permValueForChoice(best, g)
		for _, c := range candidates[1:] {
			val := permValueForChoice(c, g)
			if val > bestVal {
				best = c
				bestVal = val
			}
		}
		return best
	}
	return candidates[0]
}

// permValueForChoice scores a permanent for sacrifice/destroy choices.
// Uses current P/T for creatures (via *Game), CMC for non-creatures.
func permValueForChoice(p *Permanent, g GameReader) int {
	if p.HasType(TypeCreature) {
		return p.CurrentPower(g) + p.CurrentToughness(g)
	}
	return p.Card.ManaCost().CMC()
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

// ChooseCardFromHand picks the lowest-CMC candidate (cheapest to discard).
func (sp *SearchPlayer) ChooseCardFromHand(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.ManaCost().CMC() < best.ManaCost().CMC() {
			best = c
		}
	}
	return best
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

// ChooseCardFromLibrary prefers non-land cards over lands (tutoring for spells
// is usually more impactful). Among non-lands, picks the highest CMC card.
// Among lands, picks the first available.
func (sp *SearchPlayer) ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) == 0 {
		return nil
	}

	var bestNonLand Card
	bestCMC := -1
	for _, c := range candidates {
		if !c.HasType(TypeLand) {
			cmc := c.ManaCost().CMC()
			if cmc > bestCMC {
				bestNonLand = c
				bestCMC = cmc
			}
		}
	}
	if bestNonLand != nil {
		return bestNonLand
	}
	// All candidates are lands; return the first.
	return candidates[0]
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
