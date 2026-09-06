package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Per-turn tracking state is owned by TrackerSystem (TurnTrackers).
// The methods below provide backward-compatible Game facades and event wiring.

// recordPerTurnEvent updates per-turn trackers from a fired event. Called
// at the top of FireEvent.
func (g *Game) recordPerTurnEvent(evt *GameEvent) {
	g.trackers.Turn.RecordEvent(evt)
}

// PlayerCardsDrawnThisTurn returns the number of cards the given player has
// drawn this turn.
func (g *Game) PlayerCardsDrawnThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.PlayerCardsDrawn(playerID)
}

// PlayerCardsLeftGraveyardThisTurn returns the number of cards that left the
// given player's graveyard this turn.
func (g *Game) PlayerCardsLeftGraveyardThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.PlayerCardsLeftGraveyard(playerID)
}

// PlayerHadCardLeaveGraveyardThisTurn reports whether at least one card left
// the given player's graveyard this turn.
func (g *Game) PlayerHadCardLeaveGraveyardThisTurn(playerID uuid.UUID) bool {
	return g.trackers.Turn.PlayerHadCardLeaveGraveyard(playerID)
}

// recordCardPutIntoExile updates the global per-turn exile count.
func (g *Game) recordCardPutIntoExile(card Card) {
	g.trackers.Turn.RecordCardPutIntoExile(card)
}

func (g *Game) consumePendingExileZoneChange(cardID uuid.UUID) bool {
	return g.trackers.Turn.consumePendingExileZoneChange(cardID)
}

// CardsPutIntoExileThisTurn returns the number of cards put into exile this turn.
func (g *Game) CardsPutIntoExileThisTurn() int {
	return g.trackers.Turn.CardsPutIntoExile()
}

// PlayerDiscardCountThisTurn returns the number of times the given player has discarded a card this turn.
func (g *Game) PlayerDiscardCountThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.PlayerDiscardCount(playerID)
}

// PlayerLifeGainedThisTurn returns the total amount of life the player has gained this turn.
func (g *Game) PlayerLifeGainedThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.PlayerLifeGained(playerID)
}

// PermanentDamageReceivedThisTurn returns the total damage dealt to the permanent (or player) this turn.
func (g *Game) PermanentDamageReceivedThisTurn(permID uuid.UUID) int {
	return g.trackers.Turn.PermanentDamageReceived(permID)
}

// PermanentAttackedOrBlockedThisTurn reports whether the named permanent has either declared an attack or block this turn.
func (g *Game) PermanentAttackedOrBlockedThisTurn(permID uuid.UUID) bool {
	return g.trackers.Turn.PermanentAttackedOrBlocked(permID)
}

// PlayerCastSpellThisTurn reports whether the given player has cast at least one spell this turn.
func (g *Game) PlayerCastSpellThisTurn(playerID uuid.UUID) bool {
	return g.trackers.Turn.PlayerCastSpell(playerID)
}

// PlayerAttackedThisTurn reports whether the given player declared at least one attacker this turn.
func (g *Game) PlayerAttackedThisTurn(playerID uuid.UUID) bool {
	return g.trackers.Turn.PlayerAttacked(playerID)
}
