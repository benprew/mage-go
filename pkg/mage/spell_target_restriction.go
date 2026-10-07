package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type spellTargetRestrictionRule struct {
	subject PermanentRef
	filter  CardFilter
}

// CantBeTargetedBySpells prevents matching spells from targeting the current subject.
func CantBeTargetedBySpells(subject PermanentRef, filter CardFilter) StaticRule {
	return spellTargetRestrictionRule{subject: subject, filter: filter}
}

func (spellTargetRestrictionRule) staticRule() {}

func (r spellTargetRestrictionRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "matching spells cannot target the subject", battlefieldSource: true}
	if nilDefinition(r.subject) {
		n.problem = "spell target restriction requires a permanent reference"
	} else {
		n.children = []definitionEdge{{"subject", r.subject}}
	}
	if r.filter.IsZero() {
		n.problem = "spell target restriction requires a filter"
	}
	return n
}

func (r spellTargetRestrictionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&spellTargetRestriction{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		subject:              r.subject, filter: r.filter,
	}}
}

type spellTargetRestriction struct {
	*funcContinuousEffect
	subject PermanentRef
	filter  CardFilter
}

func (e *spellTargetRestriction) inspectDefinition() definitionNode {
	return spellTargetRestrictionRule{subject: e.subject, filter: e.filter}.inspectDefinition()
}

func (e *spellTargetRestriction) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

func (g *Game) restrictedSpellTarget(p *Permanent, spell Card) bool {
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*spellTargetRestriction)
		if !ok || nilDefinition(rule.subject) || !rule.filter.Match(spell) {
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
