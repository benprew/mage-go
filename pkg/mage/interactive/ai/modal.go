package ai

import (
	"strings"

	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// ChooseModeHeuristic selects a mode for a modal spell based on the mode
// descriptions and the player's current game state (life total, hand size,
// board state).
func ChooseModeHeuristic(modes []string, life, handSize int) int {
	bestMode := 0
	bestScore := -1

	for i, mode := range modes {
		lower := strings.ToLower(mode)
		score := scoreModeText(lower, life, handSize)

		if score > bestScore {
			bestScore = score
			bestMode = i
		}
	}

	return bestMode
}

// ChooseModeWithContext selects a mode using full game state for board-relative
// evaluation. Falls back to text heuristic when game context is unavailable.
func ChooseModeWithContext(modes []string, g *mage.Game, playerID uuid.UUID) int {
	me := g.GetPlayer(playerID)
	if me == nil {
		return ChooseModeHeuristic(modes, 20, 3)
	}

	opponent := g.GetOpponent(playerID)
	life := me.Life()
	handSize := len(me.Hand())

	// Count board state for context.
	myCreatures, oppCreatures := 0, 0
	for _, perm := range g.AllBattlefield() {
		if !perm.HasType(core.TypeCreature) {
			continue
		}
		if perm.Controller == playerID {
			myCreatures++
		} else if opponent != nil && perm.Controller == opponent.PlayerID() {
			oppCreatures++
		}
	}

	bestMode := 0
	bestScore := -1

	for i, mode := range modes {
		lower := strings.ToLower(mode)
		score := scoreModeText(lower, life, handSize)

		// Board-relative adjustments.
		if strings.Contains(lower, "destroy") || strings.Contains(lower, "exile") {
			if oppCreatures > myCreatures {
				score += 3
			}
			if oppCreatures == 0 {
				score -= 4 // no targets
			}
		}

		if strings.Contains(lower, "token") || strings.Contains(lower, "creature") {
			if strings.Contains(lower, "create") || strings.Contains(lower, "put") {
				if myCreatures < oppCreatures {
					score += 2
				}
			}
		}

		if strings.Contains(lower, "damage") || strings.Contains(lower, "deal") {
			if opponent != nil && opponent.Life() <= 5 {
				score += 3 // closer to lethal
			}
		}

		if score > bestScore {
			bestScore = score
			bestMode = i
		}
	}

	return bestMode
}

// scoreModeText scores a mode based on text content and basic game state.
func scoreModeText(lower string, life, handSize int) int {
	score := 0

	switch {
	case strings.Contains(lower, "destroy") || strings.Contains(lower, "exile"):
		score = 8
	case strings.Contains(lower, "damage") || strings.Contains(lower, "deal"):
		score = 7
	case strings.Contains(lower, "draw"):
		score = 6
		if handSize < 3 {
			score = 9
		}
	case strings.Contains(lower, "return") && strings.Contains(lower, "hand"):
		score = 5 // bounce
	case strings.Contains(lower, "counter"):
		score = 7
	case strings.Contains(lower, "gain") && strings.Contains(lower, "life"):
		score = 4
		if life <= 10 {
			score = 9
		}
	case strings.Contains(lower, "prevent"):
		score = 3
		if life <= 10 {
			score = 5
		}
	case strings.Contains(lower, "token") || strings.Contains(lower, "create"):
		score = 5
	case strings.Contains(lower, "+") || strings.Contains(lower, "gets"):
		score = 4
	default:
		score = 1
	}

	return score
}

// ChooseModeForAI picks a mode using game context when available.
func ChooseModeForAI(modes []string, g *mage.Game, playerID uuid.UUID) int {
	if g != nil {
		return ChooseModeWithContext(modes, g, playerID)
	}
	return ChooseModeHeuristic(modes, 20, 3)
}

// Ensure packages are used.
var (
	_ = eval.EvalCreatureInGame
	_ uuid.UUID
)
