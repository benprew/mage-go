package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type abilityTargetRestrictionRule struct {
	subject PermanentRef
	filter  CardFilter
}

// CantBeTargetedByAbilities prevents abilities whose sources match the filter
// from targeting the current subject. Spell targets use a separate rule.
func CantBeTargetedByAbilities(subject PermanentRef, filter CardFilter) StaticRule {
	return abilityTargetRestrictionRule{subject: subject, filter: filter}
}

func (abilityTargetRestrictionRule) staticRule() {}

func (r abilityTargetRestrictionRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "abilities from matching sources cannot target the subject", battlefieldSource: true}
	if nilDefinition(r.subject) {
		n.problem = "ability target restriction requires a permanent reference"
	} else {
		n.children = []definitionEdge{{"subject", r.subject}}
	}
	if r.filter.IsZero() {
		n.problem = "ability target restriction requires a source filter"
	}
	return n
}

func (r abilityTargetRestrictionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&abilityTargetRestriction{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), subject: r.subject, filter: r.filter,
	}}
}

type abilityTargetRestriction struct {
	*funcContinuousEffect
	subject PermanentRef
	filter  CardFilter
}

func (e *abilityTargetRestriction) inspectDefinition() definitionNode {
	return abilityTargetRestrictionRule{subject: e.subject, filter: e.filter}.inspectDefinition()
}

func (e *abilityTargetRestriction) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

func (g *Game) restrictedAbilityTarget(p *Permanent, sourceCard Card) bool {
	if sourceCard == nil {
		return false
	}
	if view, ok := sourceCard.(targetSourceLKI); ok && view.spell {
		return false
	}
	normalized := false
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*abilityTargetRestriction)
		if !ok || nilDefinition(rule.subject) {
			continue
		}
		if !normalized {
			if _, saved := sourceCard.(targetSourceLKI); !saved {
				if source := g.FindPermanentIncludingPhased(sourceCard.ID()); source != nil {
					sourceCard = targetSourceLKI{Card: damageSourceCardFromPermanent(source), colors: source.Colors(), incarnation: source.incarnationID}
				}
			}
			normalized = true
		}
		if !rule.filter.Match(sourceCard) {
			continue
		}
		source := g.FindPermanent(rule.SourceID())
		if source == nil {
			continue
		}
		ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID}
		subject, _ := rule.subject.resolvePermanent(ctx)
		if subject != nil && subject.ID() == p.ID() && subject.incarnationID == p.incarnationID {
			return true
		}
	}
	return false
}
