package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AnimateEach adds the creature type and sets base power and toughness.
// Omit colors to keep current colors. A color list replaces current colors.
func AnimateEach(subjects PermanentSet, power, toughness int, colors ...core.Color) StaticRule {
	return &staticAnimationRule{subjects: subjects, power: power, toughness: toughness, colors: modificationColors(colors), setColors: colors != nil}
}

type staticAnimationRule struct {
	subjects         PermanentSet
	power, toughness int
	colors           []core.Color
	setColors        bool
}

func (*staticAnimationRule) staticRule() {}
func (r *staticAnimationRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-animation", children: []definitionEdge{{"subjects", r.subjects}}}
}
func (r *staticAnimationRule) staticEffects() []ContinuousEffect {
	layers := []core.Layer{core.LayerType, core.LayerPT}
	if r.setColors {
		layers = []core.Layer{core.LayerType, core.LayerColor, core.LayerPT}
	}
	effects := make([]ContinuousEffect, len(layers))
	for i, layer := range layers {
		effects[i] = &staticAnimationLayer{rule: r, layer: layer}
	}
	return effects
}

type animationCycleKey struct {
	rule   *staticAnimationRule
	source uuid.UUID
}

type staticAnimationLayer struct {
	effectSource
	rule  *staticAnimationRule
	layer core.Layer
}

func (e *staticAnimationLayer) Text() string { return "animate permanent set" }
func (e *staticAnimationLayer) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *staticAnimationLayer) GetLayer() core.Layer         { return e.layer }
func (e *staticAnimationLayer) GetSublayer() core.SublayerPT { return core.SublayerPT_Set }
func (e *staticAnimationLayer) GetDuration() core.Duration   { return core.WhileOnBattlefield }
func (e *staticAnimationLayer) key() animationCycleKey       { return animationCycleKey{e.rule, e.sourceID} }
func (e *staticAnimationLayer) continuesAcrossLayers(g *Game) bool {
	_, started := g.effects.animationMembers[e.key()]
	return e.layer != core.LayerType && started
}
func (e *staticAnimationLayer) IsActive(g *Game) bool {
	return e.continuesAcrossLayers(g) || g.FindPermanent(e.sourceID) != nil
}
func (e *staticAnimationLayer) inspectDefinition() definitionNode { return e.rule.inspectDefinition() }
func (e *staticAnimationLayer) cloneEffect() ContinuousEffect     { clone := *e; return &clone }
func (e *staticAnimationLayer) selection(g *Game) []capturedPermanent {
	source := g.FindPermanent(e.sourceID)
	if source == nil {
		return nil
	}
	ctx := &EffectContext{Game: g, SourceID: e.sourceID, Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	members := []capturedPermanent{}
	for _, p := range e.rule.subjects.resolvePermanents(ctx) {
		members = append(members, capturedPermanent{p.ID(), p.incarnationID})
	}
	slices.SortFunc(members, func(a, b capturedPermanent) int { return slices.Compare(a.id[:], b.id[:]) })
	return members
}

func (e *staticAnimationLayer) DependsOn(other ContinuousEffect, g *Game) bool {
	if e.layer != core.LayerType || other.GetLayer() != core.LayerType {
		return false
	}
	// A type change can change the selection before this effect starts.
	probe := g.Clone()
	probe.effects.animationMembers = make(map[animationCycleKey][]capturedPermanent)
	trial := other
	if cloneable, ok := other.(deepCloneableEffect); ok {
		trial = cloneable.cloneEffect()
	}
	if err := trial.ApplyContinuous(&EffectContext{Game: probe}); err != nil {
		return false
	}
	probe.effects.syncAttrDeltas(probe)
	return !slices.Equal(e.selection(g), e.selection(probe))
}

func (e *staticAnimationLayer) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if e.layer == core.LayerType {
		source := g.FindPermanent(e.sourceID)
		if source == nil {
			return nil
		}
		g.effects.animationMembers[e.key()] = e.selection(g)
	}
	for _, ref := range g.effects.animationMembers[e.key()] {
		p := g.FindPermanent(ref.id)
		if p == nil || p.incarnationID != ref.incarnation {
			continue
		}
		p = g.MutablePermanent(p.ID())
		switch e.layer {
		case core.LayerType:
			for _, attr := range []core.Attr{core.AttrIsCreature, core.AttrHasPowerToughness} {
				g.effects.GrantAttr(p.ID(), attr)
			}
		case core.LayerColor:
			colors := append([]core.Color(nil), e.rule.colors...)
			p.ColorOverride = &colors
		case core.LayerPT:
			p.BasePTOverride = &[2]int{e.rule.power, e.rule.toughness}
		}
	}
	return nil
}
