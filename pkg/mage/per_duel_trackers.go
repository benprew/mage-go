package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Per-duel objective tracking is owned by TrackerSystem (DuelTrackers).
// The methods below provide backward-compatible Game facades and event wiring.

// DuelObjectives is a per-player snapshot of objective-relevant actions taken
// over the course of a duel, as seen from one player's perspective.
type DuelObjectives struct {
	SpellsByColor              map[Color]int
	SpellsByType               map[CardType]int
	LandsPlayed                int
	AttackersDeclared          int
	NonCombatDamageDealt       int // non-combat damage this player dealt to the opposing player
	OpponentCreaturesDestroyed int // the opponent's creatures that died this duel
}

// recordPerDuelEvent updates the per-duel objective counters from a fired event.
func (g *Game) recordPerDuelEvent(evt *GameEvent) {
	g.trackers.Duel.RecordEvent(evt, g.FindCardAnywhere, func(id uuid.UUID) uuid.UUID {
		if opp := g.GetOpponent(id); opp != nil {
			return opp.PlayerID()
		}
		return uuid.Nil
	})
}

// recordCreatureDeath records that a creature controlled by controllerID died this duel.
func (g *Game) recordCreatureDeath(controllerID uuid.UUID) {
	g.trackers.Duel.RecordCreatureDeath(controllerID)
}

// DuelObjectivesFor returns the per-duel objective tally from playerID's perspective.
func (g *Game) DuelObjectivesFor(playerID uuid.UUID) DuelObjectives {
	oppID := uuid.Nil
	if opp := g.GetOpponent(playerID); opp != nil {
		oppID = opp.PlayerID()
	}
	return g.trackers.Duel.ObjectivesFor(playerID, oppID)
}
