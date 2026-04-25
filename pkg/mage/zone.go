package mage

import "github.com/google/uuid"

// Functions related to zones in the game

func countBattlefield(g *Game, playerID uuid.UUID) int {
	count := 0
	for _, p := range g.battlefield {
		if p.Controller == playerID {
			count++
		}
	}
	return count
}
