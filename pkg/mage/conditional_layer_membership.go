package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WhileWithSharedSubjects keeps each selected object through all rule layers.
// Each new effect cycle reads the condition and subjects again.
func WhileWithSharedSubjects(condition StateCondition, modifications ...ContinuousModification) StaticRule {
	return &continuousRule{condition: condition, modifications: append([]ContinuousModification(nil), modifications...), sharedSubjects: true}
}

type composedCycleKey struct {
	rule   *continuousRule
	source uuid.UUID
}

type composedCycleSelection struct {
	active  bool
	members map[*layerModification][]capturedPermanent
}

func (e *composedLayerEffect) cycleKey() composedCycleKey {
	return composedCycleKey{e.rule, e.sourceID}
}

// A rule that starts in an earlier layer must keep its selected objects.
func (e *composedLayerEffect) continuesAcrossLayers(g *Game) bool {
	if e.rule == nil || !e.rule.sharedSubjects {
		return false
	}
	selection, found := g.effects.composedMembers[e.cycleKey()]
	if !found || !selection.active {
		return false
	}
	for _, modification := range e.rule.modifications {
		m := modification.modification()
		if m.layer < e.layer {
			return true
		}
		if m.layer == core.LayerPT && e.layer == core.LayerPT && ((m.setBasePT && e.GetSublayer() > core.SublayerPT_Set) || (!m.isSwitch && e.GetSublayer() == core.SublayerPT_Switch)) {
			return true
		}
	}
	return false
}

func (e *composedLayerEffect) selectMembers(g *Game) composedCycleSelection {
	result := composedCycleSelection{members: make(map[*layerModification][]capturedPermanent)}
	source := g.FindPermanent(e.sourceID)
	if source == nil || source.PhasedOut {
		return result
	}
	ctx := &EffectContext{Game: g, SourceID: e.sourceID, Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	result.active = e.rule.condition.checkState(g, StateContext{SourceID: e.sourceID, ControllerID: ctx.Controller, SourceIncarnation: source.incarnationID})
	if !result.active {
		return result
	}
	for _, modification := range e.rule.modifications {
		m := modification.modification()
		if _, seen := result.members[m]; seen {
			continue
		}
		result.members[m] = nil
		for _, p := range m.members(ctx) {
			result.members[m] = append(result.members[m], capturedPermanent{p.ID(), p.incarnationID})
		}
		slices.SortFunc(result.members[m], func(a, b capturedPermanent) int { return slices.Compare(a.id[:], b.id[:]) })
	}
	return result
}

func (e *composedLayerEffect) DependsOn(other ContinuousEffect, g *Game) bool {
	if e.rule == nil || !e.rule.sharedSubjects || e.layer != core.LayerType || other.GetLayer() != core.LayerType {
		return false
	}
	probe := probeContinuousEffect(other, g)
	if probe == nil {
		return false
	}
	before, after := e.selectMembers(g), e.selectMembers(probe)
	if before.active != after.active {
		return true
	}
	for m, members := range before.members {
		if !slices.Equal(members, after.members[m]) {
			return true
		}
	}
	for m, members := range after.members {
		if !slices.Equal(members, before.members[m]) {
			return true
		}
	}
	return false
}
