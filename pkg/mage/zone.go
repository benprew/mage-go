package mage

import "uuid"

// Functions related to zones in the game

func countBattlefield(g *Game, playerID uuid.UUID) int {
	count := 0
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == playerID {
			count++
		}
	}
	return count
}
