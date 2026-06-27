package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Per-duel objective tracking. These counters accumulate for the whole game
// (one duel) and are NOT reset between turns. They are maintained passively by
// recordPerDuelEvent — every GameEvent flows through it via FireEvent — plus
// recordCreatureDeath at the creature-death sites. They exist to let the s30
// quest system read a compact per-player tally when the duel ends, with clean
// attribution (spell color/type, exact attacker counts) that a snapshot diff
// can't provide.

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
// Called at the top of FireEvent (after recordPerTurnEvent).
func (g *Game) recordPerDuelEvent(evt *GameEvent) {
	switch evt.Type {
	case EvtSpellCast:
		g.recordSpellCastObjective(evt)
	case EvtLandPlayed:
		if g.duelLandsPlayed == nil {
			g.duelLandsPlayed = make(map[uuid.UUID]int)
		}
		g.duelLandsPlayed[evt.PlayerID]++
	case EvtDeclaredAttacker:
		if g.duelAttackersDeclared == nil {
			g.duelAttackersDeclared = make(map[uuid.UUID]int)
		}
		// PlayerID is the attacking (active) player; one event per attacker.
		g.duelAttackersDeclared[evt.PlayerID]++
	case EvtDamageDealt:
		// Only non-combat damage dealt to a player counts as "direct damage".
		if evt.Flag || evt.Amount <= 0 {
			return
		}
		target := g.GetPlayer(evt.TargetID)
		if target == nil {
			return // damage to a permanent, not a player
		}
		dealer := g.GetOpponent(evt.TargetID)
		if dealer == nil {
			return
		}
		if g.duelNonCombatDamage == nil {
			g.duelNonCombatDamage = make(map[uuid.UUID]int)
		}
		g.duelNonCombatDamage[dealer.PlayerID()] += evt.Amount
	}
}

func (g *Game) recordSpellCastObjective(evt *GameEvent) {
	card := g.FindCardAnywhere(evt.SourceID)
	if card == nil {
		return
	}
	if g.duelSpellsCastByColor == nil {
		g.duelSpellsCastByColor = make(map[uuid.UUID]map[Color]int)
	}
	byColor := g.duelSpellsCastByColor[evt.PlayerID]
	if byColor == nil {
		byColor = make(map[Color]int)
		g.duelSpellsCastByColor[evt.PlayerID] = byColor
	}
	for _, c := range card.ManaCost().Colors() {
		byColor[c]++
	}

	if g.duelSpellsCastByType == nil {
		g.duelSpellsCastByType = make(map[uuid.UUID]map[CardType]int)
	}
	byType := g.duelSpellsCastByType[evt.PlayerID]
	if byType == nil {
		byType = make(map[CardType]int)
		g.duelSpellsCastByType[evt.PlayerID] = byType
	}
	for _, t := range []CardType{TypeCreature, TypeInstant, TypeSorcery, TypeArtifact, TypeEnchantment, TypePlaneswalker} {
		if card.HasType(t) {
			byType[t]++
		}
	}
}

// recordCreatureDeath records that a creature controlled by controllerID died
// this duel. Called from the creature-death sites alongside
// creatureDeathsThisTurn++ so token deaths are counted too.
func (g *Game) recordCreatureDeath(controllerID uuid.UUID) {
	if g.duelCreatureDeaths == nil {
		g.duelCreatureDeaths = make(map[uuid.UUID]int)
	}
	g.duelCreatureDeaths[controllerID]++
}

// DuelObjectivesFor returns the per-duel objective tally from playerID's
// perspective. OpponentCreaturesDestroyed reflects the opposing player's
// creatures that died (i.e. the ones playerID is credited with destroying).
func (g *Game) DuelObjectivesFor(playerID uuid.UUID) DuelObjectives {
	obj := DuelObjectives{
		SpellsByColor:        make(map[Color]int),
		SpellsByType:         make(map[CardType]int),
		LandsPlayed:          g.duelLandsPlayed[playerID],
		AttackersDeclared:    g.duelAttackersDeclared[playerID],
		NonCombatDamageDealt: g.duelNonCombatDamage[playerID],
	}
	maps.Copy(obj.SpellsByColor, g.duelSpellsCastByColor[playerID])
	maps.Copy(obj.SpellsByType, g.duelSpellsCastByType[playerID])
	if opp := g.GetOpponent(playerID); opp != nil {
		obj.OpponentCreaturesDestroyed = g.duelCreatureDeaths[opp.PlayerID()]
	}
	return obj
}
