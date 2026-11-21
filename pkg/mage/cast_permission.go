package mage

import (
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CastAsThoughHadFlash gives the source's controller permission to cast cards
// that match the filter as though they had flash. A zero filter matches all
// cards. The permission expires when the source leaves the battlefield.
func CastAsThoughHadFlash(filter ObjectFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		perm := g.FindPermanent(sourceID)
		if perm == nil {
			return nil
		}
		g.effects.Rules.AddFlashGrant(sourceID, perm.ControllerID(), filter)
		return nil
	})
}

// CastAsThoughHadFlashGlobal gives the flash permission to all players.
// The filter uses the source permanent's controller.
func CastAsThoughHadFlashGlobal(filter ObjectFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		if g.FindPermanent(sourceID) == nil {
			return nil
		}
		g.effects.Rules.AddFlashGrant(sourceID, uuid.Nil(), filter)
		return nil
	})
}
