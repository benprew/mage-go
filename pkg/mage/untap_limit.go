package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type untapLimitRule struct {
	filter      PermanentFilter
	maximum     int
	condition   StateCondition
	conditional bool
}

// LimitUntapsDuringStep permits a choice within the limit before simultaneous untap.
// Other untap effects do not use this limit.
func LimitUntapsDuringStep(filter PermanentFilter, maximum int) StaticRule {
	return untapLimitRule{filter: filter, maximum: maximum}
}

// LimitUntapsDuringStepIf checks the condition before the untap selection.
// Untapping the source cannot change the saved selection.
func LimitUntapsDuringStepIf(filter PermanentFilter, maximum int, condition StateCondition) StaticRule {
	return untapLimitRule{filter: filter, maximum: maximum, condition: condition, conditional: true}
}

func (untapLimitRule) staticRule() {}
func (r untapLimitRule) inspectDefinition() definitionNode {
	node := definitionNode{Role: "static-rule", Text: "untap limit", battlefieldSource: true}
	if r.conditional {
		node.children = []definitionEdge{{"condition", r.condition}}
	}
	if r.maximum < 0 {
		node.problem = "untap limit must not be negative"
	}
	return node
}
func (r untapLimitRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&untapLimitEffect{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), rule: r}}
}

type untapLimitEffect struct {
	*funcContinuousEffect
	rule untapLimitRule
}

func (e *untapLimitEffect) inspectDefinition() definitionNode { return e.rule.inspectDefinition() }
func (e *untapLimitEffect) cloneEffect() ContinuousEffect {
	copyEffect := *e
	inner := *e.funcContinuousEffect
	copyEffect.funcContinuousEffect = &inner
	return &copyEffect
}

type untapQuota struct {
	members   map[uuid.UUID]bool
	remaining int
}

func (g *Game) chooseStepUntaps(active Player, blocked map[uuid.UUID]bool) []*Permanent {
	var rules []untapLimitRule
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		if limit, ok := inner.(*untapLimitEffect); ok {
			if limit.rule.conditional {
				source := g.FindPermanent(limit.SourceID())
				if source == nil || nilDefinition(limit.rule.condition) || !limit.rule.condition.checkState(g, StateContext{SourceID: source.ID(), ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID}) {
					continue
				}
			}
			rules = append(rules, limit.rule)
		}
	}
	for _, legacy := range []untapLimitRule{{filter: IsLand, maximum: g.effects.Rules.LandUntapMax}, {filter: IsArtifact, maximum: g.effects.Rules.ArtifactUntapMax}, {filter: IsCreature, maximum: g.effects.Rules.CreatureUntapMax}} {
		if legacy.maximum >= 0 {
			rules = append(rules, legacy)
		}
	}
	quotas := make([]untapQuota, len(rules))
	for i, rule := range rules {
		quotas[i] = untapQuota{members: make(map[uuid.UUID]bool), remaining: rule.maximum}
		for _, p := range g.FilterBattlefield(rule.filter) {
			quotas[i].members[p.ID()] = true
		}
	}
	var selected, candidates []*Permanent
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || p.ControllerID() != active.PlayerID() || !p.Tapped || blocked[p.ID()] || p.HasAttr(core.AttrDoesNotUntap) {
			continue
		}
		if p.HasAttr(core.AttrMayNotUntap) {
			hasTapEffect, targetLives := g.effects.SourceTapMaintainedStatus(g, p.ID())
			if (!hasTapEffect || targetLives) && !active.ChooseMayAbility("untap "+p.Name()) {
				continue
			}
		}
		limited := false
		for _, quota := range quotas {
			limited = limited || quota.members[p.ID()]
		}
		if limited {
			candidates = append(candidates, p)
		} else {
			selected = append(selected, p)
		}
	}
	for len(candidates) > 0 {
		legal := make([]*Permanent, 0, len(candidates))
		for _, p := range candidates {
			allowed := true
			for _, quota := range quotas {
				if quota.members[p.ID()] && quota.remaining <= 0 {
					allowed = false
					break
				}
			}
			if allowed {
				legal = append(legal, p)
			}
		}
		if len(legal) == 0 || !active.ChooseMayAbility("untap another permanent within the limits") {
			break
		}
		choice := active.ChoosePermanent(legal, "choose a permanent to untap", g)
		if choice == nil {
			break
		}
		index := -1
		for i, p := range legal {
			if choice.Card != nil && choice.ID() == p.ID() {
				index = i
				break
			}
		}
		if index < 0 {
			index = 0
		}
		chosen := legal[index]
		selected = append(selected, chosen)
		candidates = legal[:index:index]
		candidates = append(candidates, legal[index+1:]...)
		for i := range quotas {
			if quotas[i].members[chosen.ID()] {
				quotas[i].remaining--
			}
		}
	}
	return selected
}
