package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
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
	case EvtDiscard:
		if g.discardCountThisTurn == nil {
			g.discardCountThisTurn = make(map[uuid.UUID]int)
		}
		g.discardCountThisTurn[evt.PlayerID]++
	case EvtLifeGained:
		if g.lifeGainedThisTurn == nil {
			g.lifeGainedThisTurn = make(map[uuid.UUID]int)
		}
		amt := evt.Amount
		if amt < 0 {
			amt = 0
		}
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
		amt := evt.Amount
		if amt < 0 {
			amt = 0
		}
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
