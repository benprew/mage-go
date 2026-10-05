package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type basicLandSetEffect struct {
	effectSource
	filter   PermanentFilter
	subtypes []string
}

func (*basicLandSetEffect) Properties() EffectProperties { return EffectProperties{} }
func (*basicLandSetEffect) Text() string                 { return "set basic land types" }
func (*basicLandSetEffect) GetLayer() core.Layer         { return core.LayerType }
func (*basicLandSetEffect) GetDuration() core.Duration   { return core.WhileOnBattlefield }
func (e *basicLandSetEffect) IsActive(g *Game) bool      { return g.FindPermanent(e.sourceID) != nil }
func (e *basicLandSetEffect) selection(g *Game) []uuid.UUID {
	var ids []uuid.UUID
	for _, p := range g.FilterBattlefield(e.filter) {
		if p.HasType(core.TypeLand) {
			ids = append(ids, p.ID())
		}
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	return ids
}
func (e *basicLandSetEffect) DependsOn(other ContinuousEffect, g *Game) bool {
	if other.GetLayer() != core.LayerType {
		return false
	}
	probe := probeContinuousEffect(other, g)
	if probe == nil {
		return false
	}
	return !slices.Equal(e.selection(g), e.selection(probe))
}
func (e *basicLandSetEffect) ApplyContinuous(ctx *EffectContext) error {
	for _, id := range e.selection(ctx.Game) {
		if p := ctx.Game.MutablePermanent(id); p != nil {
			applyBecomesBasicLand(ctx.Game, p, e.subtypes)
		}
	}
	return nil
}

// A trial must not change the live game or another trial.
func probeContinuousEffect(effect ContinuousEffect, g *Game) *Game {
	probe := g.Clone()
	probe.effects.animationMembers = make(map[animationCycleKey][]capturedPermanent)
	trial := effect
	if cloneable, ok := effect.(deepCloneableEffect); ok {
		trial = cloneable.cloneEffect()
	}
	if err := trial.ApplyContinuous(&EffectContext{Game: probe}); err != nil {
		return nil
	}
	probe.effects.syncAttrDeltas(probe)
	return probe
}
