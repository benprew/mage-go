package ai

import "strings"

// ChooseModeHeuristic selects a mode for a modal spell based on the mode
// descriptions and the player's current game state (life total, hand size).
func ChooseModeHeuristic(modes []string, life, handSize int) int {
	bestMode := 0
	bestScore := -1

	for i, mode := range modes {
		lower := strings.ToLower(mode)
		score := 0

		switch {
		case strings.Contains(lower, "destroy"):
			score = 8
		case strings.Contains(lower, "damage") || strings.Contains(lower, "deal"):
			score = 7
		case strings.Contains(lower, "draw"):
			score = 6
			if handSize < 3 {
				score = 9
			}
		case strings.Contains(lower, "gain") && strings.Contains(lower, "life"):
			score = 4
			if life <= 10 {
				score = 9
			}
		case strings.Contains(lower, "prevent"):
			score = 3
		default:
			score = 1
		}

		if score > bestScore {
			bestScore = score
			bestMode = i
		}
	}

	return bestMode
}
