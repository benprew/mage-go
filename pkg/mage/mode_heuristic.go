package mage

import "strings"

// ChooseModeHeuristic picks the best mode index based on mode text and game state.
// Used by both AIPlayer and SearchPlayer for intelligent mode selection.
//
// Heuristics (in priority order):
//   - "gain life" / healing modes: prefer when life < 10
//   - "deal damage" modes: prefer when life is healthy (>= 10)
//   - "draw" modes: prefer when hand size < 3
//   - "destroy" modes: always valuable (score 3)
//   - Fall back to mode 0 for unknown modes
func ChooseModeHeuristic(modes []string, life, handSize int) int {
	bestIdx := 0
	bestScore := 0

	for i, mode := range modes {
		lower := strings.ToLower(mode)
		score := 0

		switch {
		case strings.Contains(lower, "gain") && strings.Contains(lower, "life"):
			if life < 10 {
				score = 6 // high priority when low life
			} else {
				score = 1
			}
		case strings.Contains(lower, "damage") || strings.Contains(lower, "deal"):
			if life >= 10 {
				score = 5 // prefer offense when healthy
			} else {
				score = 2
			}
		case strings.Contains(lower, "draw"):
			if handSize < 3 {
				score = 4 // card draw is great when hand is empty
			} else {
				score = 2
			}
		case strings.Contains(lower, "destroy"):
			score = 3 // always decent
		case strings.Contains(lower, "prevent"):
			if life < 10 {
				score = 4
			} else {
				score = 2
			}
		}

		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	return bestIdx
}
