package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type staticDamagePreventionRule struct {
	subject    PermanentRef
	from       PermanentFilter
	outgoing   bool
	allDamage  bool
	sourceType core.CardType
}

// PreventCombatDamageFrom prevents combat damage from the current subject.
func PreventCombatDamageFrom(subject PermanentRef) StaticRule {
	return staticDamagePreventionRule{subject: subject, outgoing: true}
}

// PreventCombatDamageTo prevents matching combat damage to the current subject.
// A zero source filter permits every source.
func PreventCombatDamageTo(subject PermanentRef, from PermanentFilter) StaticRule {
	return staticDamagePreventionRule{subject: subject, from: from}
}

// PreventDamageFromCardType prevents damage from sources with the specified type.
// The rule reads the current subject so an Aura can change its attachment.
func PreventDamageFromCardType(subject PermanentRef, kind core.CardType) StaticRule {
	return staticDamagePreventionRule{subject: subject, allDamage: true, sourceType: kind}
}

func (staticDamagePreventionRule) staticRule() {}
func (r staticDamagePreventionRule) inspectDefinition() definitionNode {
	node := definitionNode{Role: "static-rule", Text: "prevent combat damage", battlefieldSource: true, children: []definitionEdge{{"subject", r.subject}}}
	if r.allDamage {
		node.Text = "prevent damage from a card type"
		if r.sourceType < core.TypeCreature || r.sourceType > core.TypePlaneswalker {
			node.problem = "supply a valid damage source card type"
		}
	}
	return node
}
func (r staticDamagePreventionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&staticDamagePreventionEffect{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), rule: r}}
}

type staticDamagePreventionEffect struct {
	*funcContinuousEffect
	rule staticDamagePreventionRule
}

func (e *staticDamagePreventionEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *staticDamagePreventionEffect) cloneEffect() ContinuousEffect {
	cloned := *e
	inner := *e.funcContinuousEffect
	cloned.funcContinuousEffect = &inner
	return &cloned
}
func (e *staticDamagePreventionEffect) ApplyContinuous(ctx *EffectContext) error {
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
	ctx.Game.effects.AddCycleReplacement(&staticDamagePreventionReplacement{
		replacementBase:   replacementBase{sourceID: source.ID()},
		sourceIncarnation: source.incarnationID,
		rule:              e.rule,
	})
	return nil
}

type staticDamagePreventionReplacement struct {
	replacementBase
	sourceIncarnation uuid.UUID
	rule              staticDamagePreventionRule
}

func (r *staticDamagePreventionReplacement) Matches(action Action, reader GameReader) bool {
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
	match := &damagePreventionRuleReplacement{combatOnly: !r.rule.allDamage}
	if r.rule.allDamage && !damageSourceHasCardType(action, g, r.rule.sourceType) {
		return false
	}
	if r.rule.outgoing {
		match.from = identity
	} else {
		match.from = r.rule.from
		match.to = identity
	}
	return match.Matches(action, reader)
}
func (*staticDamagePreventionReplacement) Replace(Action, *Game) Action { return nil }
func (*staticDamagePreventionReplacement) IsActive(GameReader) bool     { return true }
func (*staticDamagePreventionReplacement) IsPreventionEffect() bool     { return true }
func (r *staticDamagePreventionReplacement) Clone() ReplacementEffect   { cloned := *r; return &cloned }

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

func damageSourceHasCardType(action Action, g *Game, kind core.CardType) bool {
	var source DamageSource
	switch damage := action.(type) {
	case *DamageToCreatureAction:
		source = damage.DamageSource()
	case *DamageToPlayerAction:
		source = damage.DamageSource()
	default:
		return false
	}
	if permanent := g.damageFilterSource(action.ActionSource(), source); permanent != nil {
		return permanent.HasType(kind)
	}
	// A saved permanent must not use a later visit of the same card.
	return source.Valid() && source.ID() == action.ActionSource() && source.Incarnation() == uuid.Nil && source.HasType(kind)
}
