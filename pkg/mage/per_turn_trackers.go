package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Per-turn tracking state. These maps live on *Game and are reset by
// resetPerTurnState(). They are maintained passively by recordPerTurnEvent
// — every GameEvent flows through it via FireEvent.
//
// The accessors below are exposed as engine API for cards that need to
// consult "this turn" history at resolution time (Stinkdrinker Daredevil,
// Sage of Hours, Healer of the Glade, etc.).

// recordPerTurnEvent updates per-turn trackers from a fired event. Called
// at the top of FireEvent so even consumer-modified events still update
// the counters.
func (g *Game) recordPerTurnEvent(evt *GameEvent) {
	switch evt.Type {
	case EvtZoneChange:
		if evt.ToZone != ZoneExile || evt.SourceID == uuid.Nil {
			return
		}
		g.cardsPutIntoExileThisTurn++
		if g.exileZoneChangesPending == nil {
			g.exileZoneChangesPending = make(map[uuid.UUID]int)
		}
		g.exileZoneChangesPending[evt.SourceID]++
	case EvtDiscard:
		if g.discardCountThisTurn == nil {
			g.discardCountThisTurn = make(map[uuid.UUID]int)
		}
		g.discardCountThisTurn[evt.PlayerID]++
	case EvtLifeGained:
		if g.lifeGainedThisTurn == nil {
			g.lifeGainedThisTurn = make(map[uuid.UUID]int)
		}
		amt := max(evt.Amount, 0)
		g.lifeGainedThisTurn[evt.PlayerID] += amt
	case EvtDamageDealt:
		// TargetID may be a permanent or a player. Track both — callers
		// disambiguate via the accessor they call.
		if evt.TargetID == uuid.Nil {
			return
		}
		if g.permDamageReceivedThisTurn == nil {
			g.permDamageReceivedThisTurn = make(map[uuid.UUID]int)
		}
		amt := max(evt.Amount, 0)
		g.permDamageReceivedThisTurn[evt.TargetID] += amt
	case EvtDeclaredBlocker:
		if !evt.Flag {
			return
		}
		if g.attackedOrBlockedThisTurn == nil {
			g.attackedOrBlockedThisTurn = make(map[uuid.UUID]bool)
		}
		// SourceID = blocker permanent ID
		g.attackedOrBlockedThisTurn[evt.SourceID] = true
	case EvtDeclaredAttacker:
		if g.attackedOrBlockedThisTurn == nil {
			g.attackedOrBlockedThisTurn = make(map[uuid.UUID]bool)
		}
		g.attackedOrBlockedThisTurn[evt.SourceID] = true
		// Track which player declared an attacker this turn (Angelic Arbiter,
		// etc.). PlayerID on EvtDeclaredAttacker is the attacking (active)
		// player.
		if g.playerAttackedThisTurn == nil {
			g.playerAttackedThisTurn = make(map[uuid.UUID]bool)
		}
		g.playerAttackedThisTurn[evt.PlayerID] = true
	case EvtCardDrawn:
		if g.cardsDrawnThisTurn == nil {
			g.cardsDrawnThisTurn = make(map[uuid.UUID]int)
		}
		g.cardsDrawnThisTurn[evt.PlayerID]++
	case EvtCardsLeftGraveyard:
		if g.cardsLeftGraveyardThisTurn == nil {
			g.cardsLeftGraveyardThisTurn = make(map[uuid.UUID]int)
		}
		amt := evt.Amount
		if amt <= 0 {
			amt = 1
		}
		g.cardsLeftGraveyardThisTurn[evt.PlayerID] += amt
	case EvtSpellCast:
		// Track which player cast a spell this turn (Angelic Arbiter, etc.).
		// PlayerID on EvtSpellCast is the casting player.
		if g.playerCastSpellThisTurn == nil {
			g.playerCastSpellThisTurn = make(map[uuid.UUID]bool)
		}
		g.playerCastSpellThisTurn[evt.PlayerID] = true
	}
}

// resetPerTurnTrackers clears all per-turn tracker state. Called by the
// turn-end cleanup pipeline (alongside the existing damage / attacked-this-turn
// resets).
func (g *Game) resetPerTurnTrackers() {
	g.discardCountThisTurn = nil
	g.lifeGainedThisTurn = nil
	g.permDamageReceivedThisTurn = nil
	g.attackedOrBlockedThisTurn = nil
	g.playerCastSpellThisTurn = nil
	g.playerAttackedThisTurn = nil
	g.cardsDrawnThisTurn = nil
	g.cardsLeftGraveyardThisTurn = nil
	g.cardsPutIntoExileThisTurn = 0
	g.exileZoneChangesPending = nil
}

// PlayerCardsDrawnThisTurn returns the number of cards the given player has
// drawn this turn (counts every EvtCardDrawn — turn-based draws, replacement-
// granted extra draws like Howling Mine, and resolved spells/abilities).
// Used by cards that key on "their first card each turn" (Zurzoth, Chaos
// Rider).
func (g *Game) PlayerCardsDrawnThisTurn(playerID uuid.UUID) int {
	return g.cardsDrawnThisTurn[playerID]
}

// PlayerCardsLeftGraveyardThisTurn returns the number of cards that left the
// given player's graveyard this turn. Multi-card bursts are counted by the
// EvtCardsLeftGraveyard Amount.
func (g *Game) PlayerCardsLeftGraveyardThisTurn(playerID uuid.UUID) int {
	return g.cardsLeftGraveyardThisTurn[playerID]
}

// PlayerHadCardLeaveGraveyardThisTurn reports whether at least one card left
// the given player's graveyard this turn.
func (g *Game) PlayerHadCardLeaveGraveyardThisTurn(playerID uuid.UUID) bool {
	return g.PlayerCardsLeftGraveyardThisTurn(playerID) > 0
}

// recordCardPutIntoExile updates the global per-turn exile count.
func (g *Game) recordCardPutIntoExile(card Card) {
	if card == nil {
		return
	}
	if g.consumePendingExileZoneChange(card.ID()) {
		return
	}
	g.cardsPutIntoExileThisTurn++
}

func (g *Game) consumePendingExileZoneChange(cardID uuid.UUID) bool {
	if cardID == uuid.Nil || g.exileZoneChangesPending == nil {
		return false
	}
	n := g.exileZoneChangesPending[cardID]
	if n <= 0 {
		return false
	}
	if n == 1 {
		delete(g.exileZoneChangesPending, cardID)
	} else {
		g.exileZoneChangesPending[cardID] = n - 1
	}
	return true
}

// CardsPutIntoExileThisTurn returns the number of cards put into exile this
// turn, regardless of owner/controller.
func (g *Game) CardsPutIntoExileThisTurn() int {
	return g.cardsPutIntoExileThisTurn
}

// PlayerDiscardCountThisTurn returns the number of times the given player
// has discarded a card this turn (any reason — cost, effect, hand-size).
func (g *Game) PlayerDiscardCountThisTurn(playerID uuid.UUID) int {
	return g.discardCountThisTurn[playerID]
}

// PlayerLifeGainedThisTurn returns the total amount of life the player
// has gained this turn (sum of EvtLifeGained.Amount).
func (g *Game) PlayerLifeGainedThisTurn(playerID uuid.UUID) int {
	return g.lifeGainedThisTurn[playerID]
}

// PermanentDamageReceivedThisTurn returns the total damage dealt to the
// permanent (or player) this turn. Sums EvtDamageDealt.Amount across all
// damage events whose TargetID matches.
func (g *Game) PermanentDamageReceivedThisTurn(permID uuid.UUID) int {
	return g.permDamageReceivedThisTurn[permID]
}

// PermanentAttackedOrBlockedThisTurn reports whether the named permanent
// has either declared an attack or declared a block this turn. Useful for
// cards like Heart of Light or Foundry Champion.
func (g *Game) PermanentAttackedOrBlockedThisTurn(permID uuid.UUID) bool {
	return g.attackedOrBlockedThisTurn[permID]
}

// PlayerCastSpellThisTurn reports whether the given player has cast at
// least one spell this turn. Used by cards like Angelic Arbiter that key
// on whether an opponent cast a spell this turn.
func (g *Game) PlayerCastSpellThisTurn(playerID uuid.UUID) bool {
	return g.playerCastSpellThisTurn[playerID]
}

// PlayerAttackedThisTurn reports whether the given player declared at
// least one attacker this turn. Used by cards like Angelic Arbiter that
// key on whether an opponent attacked with a creature this turn.
func (g *Game) PlayerAttackedThisTurn(playerID uuid.UUID) bool {
	return g.playerAttackedThisTurn[playerID]
}
