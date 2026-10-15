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

// AnimateArtifactsUntilCleanupAfterExit animates each noncreature artifact.
// Each member loses all abilities and has base P/T equal to its mana value.
// After the source leaves, the same effect continues until cleanup.
func AnimateArtifactsUntilCleanupAfterExit() StaticRule {
	return &staticAnimationRule{subjects: MatchingPermanents(And(IsArtifact, Not(IsCreature))), artifactContinuation: true}
}

type staticAnimationRule struct {
	subjects             PermanentSet
	power, toughness     int
	colors               []core.Color
	setColors            bool
	artifactContinuation bool
}

func (*staticAnimationRule) staticRule() {}
func (r *staticAnimationRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-animation", children: []definitionEdge{{"subjects", r.subjects}}}
}
func (r *staticAnimationRule) staticEffects() []ContinuousEffect {
	layers := []core.Layer{core.LayerType, core.LayerPT}
	if r.artifactContinuation {
		layers = []core.Layer{core.LayerType, core.LayerAbility, core.LayerPT}
	}
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
	rule        *staticAnimationRule
	source      uuid.UUID
	incarnation uuid.UUID
}

type staticAnimationLayer struct {
	effectSource
	rule        *staticAnimationRule
	layer       core.Layer
	incarnation uuid.UUID
	departed    bool
}

func (e *staticAnimationLayer) Text() string { return "animate permanent set" }
func (e *staticAnimationLayer) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *staticAnimationLayer) GetLayer() core.Layer         { return e.layer }
func (e *staticAnimationLayer) GetSublayer() core.SublayerPT { return core.SublayerPT_Set }
func (e *staticAnimationLayer) GetDuration() core.Duration {
	if e.departed {
		return core.EndOfTurn
	}
	return core.WhileOnBattlefield
}
func (e *staticAnimationLayer) key() animationCycleKey {
	return animationCycleKey{e.rule, e.sourceID, e.incarnation}
}
func (e *staticAnimationLayer) continuesAcrossLayers(g *Game) bool {
	_, started := g.effects.animationMembers[e.key()]
	return e.layer != core.LayerType && started
}
func (e *staticAnimationLayer) IsActive(g *Game) bool {
	if e.departed || e.continuesAcrossLayers(g) {
		return true
	}
	p := g.FindPermanent(e.sourceID)
	return p != nil && (e.incarnation == uuid.Nil || p.incarnationID == e.incarnation)
}

func (e *staticAnimationLayer) survivesSourceExit() bool { return e.departed }

func (e *staticAnimationLayer) continueAfterSourceExit(g *Game, source *Permanent) ContinuousEffect {
	if !e.rule.artifactContinuation || e.departed || source.incarnationID != e.incarnation {
		return nil
	}
	// Layer 6 ability removal cannot cancel an effect that started in layer 4.
	if _, started := g.effects.animationMembers[e.key()]; !started {
		return nil
	}
	clone := *e
	clone.departed = true
	return &clone
}
func (e *staticAnimationLayer) inspectDefinition() definitionNode { return e.rule.inspectDefinition() }
func (e *staticAnimationLayer) cloneEffect() ContinuousEffect     { clone := *e; return &clone }
func (e *staticAnimationLayer) selection(g *Game) []capturedPermanent {
	source := g.FindPermanent(e.sourceID)
	if !e.departed && (source == nil || source.incarnationID != e.incarnation && e.incarnation != uuid.Nil) {
		return nil
	}
	ctx := &EffectContext{Game: g, SourceID: e.sourceID, sourceCaptured: true, sourceIncarnation: e.incarnation}
	if !e.departed {
		ctx.Controller = source.ControllerID()
	}
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
			if e.rule.artifactContinuation {
				g.effects.GrantAttr(p.ID(), core.AttrIsArtifact)
			}
		case core.LayerColor:
			colors := append([]core.Color(nil), e.rule.colors...)
			p.ColorOverride = &colors
		case core.LayerAbility:
			removeAllAbilitiesAtCurrentLayer(g, p)
		case core.LayerPT:
			if e.rule.artifactContinuation {
				value := p.Card.ManaCost().CMC()
				if p.FaceDown {
					value = 0
				}
				p.BasePTOverride = &[2]int{value, value}
			} else {
				p.BasePTOverride = &[2]int{e.rule.power, e.rule.toughness}
			}
		}
	}
	return nil
}
