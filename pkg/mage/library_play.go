package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// RevealTopCardOfLibrary returns a continuous effect that, while the source
// permanent is on the battlefield, marks the source's controller as playing
// with the top card of their library revealed (CR 702.X — see Future Sight,
// Oracle of Mul Daya, Magus of the Future). The flag is registered each
// Apply() cycle on the GameRules so it auto-clears when the source leaves
// play.
func RevealTopCardOfLibrary() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		perm := g.FindPermanent(sourceID)
		if perm == nil {
			return nil
		}
		g.effects.Rules.AddRevealedTopCard(perm.Controller)
		return nil
	})
}

// PlayLandsFromTopOfLibrary returns a continuous effect that lets the
// source permanent's controller play lands from the top of their library
// (CR 305.4a allows this with the appropriate static permission). The
// permission is registered each Apply() cycle on the GameRules and clears
// when the source leaves the battlefield.
func PlayLandsFromTopOfLibrary() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		perm := g.FindPermanent(sourceID)
		if perm == nil {
			return nil
		}
		g.effects.Rules.AddPlayLandsFromZone(perm.Controller, ZoneLibrary)
		return nil
	})
}

// AdditionalLandPlayStatic returns a continuous effect that grants the
// source permanent's controller +1 to their max land plays each turn while
// the source is on the battlefield (Azusa, Oracle of Mul Daya, Exploration).
// Registered on each Apply() cycle so it auto-clears with the source.
func AdditionalLandPlayStatic() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		perm := g.FindPermanent(sourceID)
		if perm == nil {
			return nil
		}
		g.effects.Rules.AddAdditionalLandPlay(perm.Controller, 1)
		return nil
	})
}
