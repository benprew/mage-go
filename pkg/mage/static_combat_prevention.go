package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type staticCombatPreventionRule struct {
	subject  PermanentRef
	from     PermanentFilter
	outgoing bool
}

// PreventCombatDamageFrom prevents combat damage from the current subject.
func PreventCombatDamageFrom(subject PermanentRef) StaticRule {
	return staticCombatPreventionRule{subject: subject, outgoing: true}
}

// PreventCombatDamageTo prevents matching combat damage to the current subject.
// A zero source filter permits every source.
func PreventCombatDamageTo(subject PermanentRef, from PermanentFilter) StaticRule {
	return staticCombatPreventionRule{subject: subject, from: from}
}

func (staticCombatPreventionRule) staticRule() {}
func (r staticCombatPreventionRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-rule", Text: "prevent combat damage", battlefieldSource: true, children: []definitionEdge{{"subject", r.subject}}}
}
func (r staticCombatPreventionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&staticCombatPreventionEffect{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), rule: r}}
}

type staticCombatPreventionEffect struct {
	*funcContinuousEffect
	rule staticCombatPreventionRule
}

func (e *staticCombatPreventionEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *staticCombatPreventionEffect) cloneEffect() ContinuousEffect {
	cloned := *e
	inner := *e.funcContinuousEffect
	cloned.funcContinuousEffect = &inner
	return &cloned
}
func (e *staticCombatPreventionEffect) ApplyContinuous(ctx *EffectContext) error {
	if nilDefinition(e.rule.subject) {
		return nil
	}
	source := ctx.Game.FindPermanent(e.SourceID())
	if source == nil {
		return nil
	}
	context := &EffectContext{Game: ctx.Game, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	_, ok := e.rule.subject.resolvePermanent(context)
	if !ok {
		return nil
	}
	ctx.Game.effects.AddCycleReplacement(&staticCombatPreventionReplacement{
		replacementBase:   replacementBase{sourceID: source.ID()},
		sourceIncarnation: source.incarnationID,
		rule:              e.rule,
	})
	return nil
}

type staticCombatPreventionReplacement struct {
	replacementBase
	sourceIncarnation uuid.UUID
	rule              staticCombatPreventionRule
}

func (r *staticCombatPreventionReplacement) Matches(action Action, reader GameReader) bool {
	// The source can phase out before the next layer cycle.
	g := reader.(*Game)
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.incarnationID != r.sourceIncarnation {
		return false
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	subject, ok := r.rule.subject.resolvePermanent(ctx)
	if !ok {
		return false
	}
	id, incarnation := subject.ID(), subject.incarnationID
	identity := NewPermanentFilter("current prevention subject", func(p *Permanent, _ *Game) bool { return p.ID() == id && p.incarnationID == incarnation })
	match := &damagePreventionRuleReplacement{combatOnly: true}
	if r.rule.outgoing {
		match.from = identity
	} else {
		match.from = r.rule.from
		match.to = identity
	}
	return match.Matches(action, reader)
}
func (*staticCombatPreventionReplacement) Replace(Action, *Game) Action { return nil }
func (*staticCombatPreventionReplacement) IsActive(GameReader) bool     { return true }
func (*staticCombatPreventionReplacement) IsPreventionEffect() bool     { return true }
func (r *staticCombatPreventionReplacement) Clone() ReplacementEffect   { cloned := *r; return &cloned }

// DamageSourceIsEnchanted reads the source's current or retained Aura status.
// A returned permanent cannot replace the departed source's information.
func DamageSourceIsEnchanted() PermanentFilter {
	return NewPermanentFilter("enchanted damage source", func(p *Permanent, g *Game) bool {
		if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
			if !current.PhasedOut {
				return IsEnchanted.Match(current, g)
			}
			information := g.zones.phaseInformation[p.incarnationID]
			return information != nil && information.Enchanted
		}
		information := g.zones.LKIByIncarnation(p.incarnationID)
		return information != nil && information.ID == p.ID() && information.Enchanted
	})
}
