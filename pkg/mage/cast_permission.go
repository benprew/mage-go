package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CastAsThoughHadFlash returns a continuous effect that grants the source
// permanent's controller permission to cast cards matching the filter "as
// though they had flash" (CR 702.8). Used by Rattlechains ("you may cast
// Spirit spells as though they had flash"), Vedalken Orrery, and
// Leyline of Anticipation (filter == nil grants every nonland card).
//
// The grant lives on the GameRules registry, registered each Apply() cycle
// while the source is on the battlefield. It expires automatically when the
// source leaves play.
func CastAsThoughHadFlash(filter CardFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		perm := g.FindPermanent(sourceID)
		if perm == nil {
			return nil
		}
		g.effects.Rules.AddFlashGrant(sourceID, perm.ControllerID(), filter)
		return nil
	})
}

// CastAsThoughHadFlashGlobal grants the flash permission to every player,
// for cards matching filter (or every nonland card if filter == nil). Used
// by Vedalken Orrery / Leyline of Anticipation.
func CastAsThoughHadFlashGlobal(filter CardFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		if g.FindPermanent(sourceID) == nil {
			return nil
		}
		g.effects.Rules.AddFlashGrant(sourceID, uuid.Nil, filter)
		return nil
	})
}
