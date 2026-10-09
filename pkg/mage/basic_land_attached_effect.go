package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type basicLandAttachedEffect struct {
	attachedEffect
	subtypes []string
	chosen   bool
}

func newBasicLandAttachedEffect(subtypes []string, chosen bool) ContinuousEffect {
	return &basicLandAttachedEffect{
		attachedEffect: attachedEffect{layer: core.LayerType, sublayer: core.SublayerPT_Modify},
		subtypes:       slices.Clone(subtypes), chosen: chosen,
	}
}

func (e *basicLandAttachedEffect) ApplyContinuous(ctx *EffectContext) error {
	if !e.IsActive(ctx.Game) {
		return nil
	}
	source := ctx.Game.FindPermanent(e.sourceID)
	target := ctx.Game.MutablePermanent(source.AttachedTo)
	subtypes := e.subtypes
	if e.chosen {
		landType := source.ChosenBasicLandType.String()
		if landType == "" {
			return nil
		}
		subtypes = []string{landType}
	}
	applyBecomesBasicLand(ctx.Game, target, subtypes)
	return nil
}

func (e *basicLandAttachedEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.subtypes = slices.Clone(e.subtypes)
	return &clone
}
