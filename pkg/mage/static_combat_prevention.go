package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type staticDamagePreventionRule struct {
	condition        StateCondition
	conditional      bool
	subjects         PermanentSet
	subject          PermanentRef
	from             PermanentFilter
	outgoing         bool
	allDamage        bool
	blockedCreatures bool
	permanentSources bool
	targetedSpells   bool
	sourceType       core.CardType
}

// PreventDamageFromMatchingPermanents prevents matching damage to the current subject.
// A zero filter permits any permanent source. Spell sources cannot match.
func PreventDamageFromMatchingPermanents(subject PermanentRef, from PermanentFilter) StaticRule {
	if from.IsZero() {
		from = AnyPermanent
	}
	return staticDamagePreventionRule{subject: subject, from: from, allDamage: true, permanentSources: true}
}

// PreventDamageFromMatchingPermanentsIf checks the condition before prevention.
// The condition uses the current source controller and battlefield visit.
func PreventDamageFromMatchingPermanentsIf(subject PermanentRef, from PermanentFilter, condition StateCondition) StaticRule {
	rule := PreventDamageFromMatchingPermanents(subject, from).(staticDamagePreventionRule)
	rule.condition, rule.conditional = condition, true
	return rule
}

// PreventDamageFromMatchingPermanentsToSetIf checks live recipients before prevention.
// The source visit and its current controller supply the query context.
func PreventDamageFromMatchingPermanentsToSetIf(subjects PermanentSet, from PermanentFilter, condition StateCondition) StaticRule {
	if from.IsZero() {
		from = AnyPermanent
	}
	return staticDamagePreventionRule{subjects: subjects, from: from, condition: condition, conditional: true, allDamage: true, permanentSources: true}
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

// PreventDamageFromSpellsTargeting prevents spell damage with a saved target visit.
func PreventDamageFromSpellsTargeting(subject PermanentRef) StaticRule {
	return staticDamagePreventionRule{subject: subject, targetedSpells: true, allDamage: true}
}

// PreventDamageFromSpellsTargetingIf checks the live condition before prevention.
func PreventDamageFromSpellsTargetingIf(subject PermanentRef, condition StateCondition) StaticRule {
	rule := PreventDamageFromSpellsTargeting(subject).(staticDamagePreventionRule)
	rule.condition, rule.conditional = condition, true
	return rule
}

// PreventDamageFromBlockedCreatures prevents damage from current blocked creatures.
// A former combat pair cannot supply this prevention through retained information.
func PreventDamageFromBlockedCreatures(subject PermanentRef) StaticRule {
	return staticDamagePreventionRule{subject: subject, allDamage: true, blockedCreatures: true}
}

func (staticDamagePreventionRule) staticRule() {}
func (r staticDamagePreventionRule) inspectDefinition() definitionNode {
	node := definitionNode{Role: "static-rule", Text: "prevent combat damage", battlefieldSource: true, children: []definitionEdge{{"subject", r.subject}}}
	if !nilDefinition(r.subjects) {
		node.children = []definitionEdge{{"subjects", r.subjects}}
	}
	if r.targetedSpells {
		node.Text = "prevent damage from spells with a matching target"
	} else if r.blockedCreatures {
		node.Text = "prevent damage from blocked creatures"
	} else if r.permanentSources {
		node.Text = "prevent damage from matching permanent sources"
	} else if r.allDamage {
		node.Text = "prevent damage from a card type"
		if r.sourceType < core.TypeCreature || r.sourceType > core.TypePlaneswalker {
			node.problem = "supply a valid damage source card type"
		}
	}
	if r.conditional {
		if nilDefinition(r.condition) {
			node.problem = "conditional prevention requires a condition"
		} else {
			node.children = append(node.children, definitionEdge{"condition", r.condition})
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
	if nilDefinition(e.rule.subject) && nilDefinition(e.rule.subjects) {
		return nil
	}
	source := ctx.Game.FindPermanent(e.SourceID())
	if source == nil {
		return nil
	}
	context := &EffectContext{Game: ctx.Game, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	if nilDefinition(e.rule.subjects) {
		_, ok := e.rule.subject.resolvePermanent(context)
		if !ok {
			return nil
		}
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

func (r *staticDamagePreventionReplacement) currentSubject(g *Game) (*Permanent, bool) {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.incarnationID != r.sourceIncarnation {
		return nil, false
	}
	if r.rule.conditional && (nilDefinition(r.rule.condition) || !r.rule.condition.checkState(g, StateContext{SourceID: source.ID(), ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID})) {
		return nil, false
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	return r.rule.subject.resolvePermanent(ctx)
}

func (r *staticDamagePreventionReplacement) currentSubjects(g *Game) []*Permanent {
	if nilDefinition(r.rule.subjects) {
		subject, ok := r.currentSubject(g)
		if !ok {
			return nil
		}
		return []*Permanent{subject}
	}
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.incarnationID != r.sourceIncarnation {
		return nil
	}
	if r.rule.conditional && (nilDefinition(r.rule.condition) || !r.rule.condition.checkState(g, StateContext{SourceID: source.ID(), ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID})) {
		return nil
	}
	return r.rule.subjects.resolvePermanents(&EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID})
}

func (r *staticDamagePreventionReplacement) Matches(action Action, reader GameReader) bool {
	g := reader.(*Game)
	subjects := r.currentSubjects(g)
	if len(subjects) == 0 {
		return false
	}
	identity := NewPermanentFilter("current prevention subjects", func(p *Permanent, _ *Game) bool {
		for _, subject := range subjects {
			if p.ID() == subject.ID() && p.incarnationID == subject.incarnationID {
				return true
			}
		}
		return false
	})
	match := &damagePreventionRuleReplacement{combatOnly: !r.rule.allDamage}
	if r.rule.targetedSpells {
		if !damageActionSource(action).TargetsPermanent(subjects[0]) {
			return false
		}
		switch damage := action.(type) {
		case *DamageToCreatureAction:
			if damage.IsCombatDamage() {
				return false
			}
		case *DamageToPlayerAction:
			if damage.IsCombatDamage() {
				return false
			}
		default:
			return false
		}
	}
	if r.rule.allDamage && !r.rule.targetedSpells && !r.rule.blockedCreatures && !r.rule.permanentSources && !damageSourceHasCardType(action, g, r.rule.sourceType) {
		return false
	}
	if r.rule.outgoing {
		match.from = identity
	} else {
		match.from = r.rule.from
		if r.rule.blockedCreatures {
			match.from = NewPermanentFilter("creature blocked by current subject", func(p *Permanent, game *Game) bool {
				current := game.FindPermanent(p.ID())
				return current != nil && current.incarnationID == p.incarnationID && current.HasType(core.TypeCreature) &&
					game.combat.BlocksIncarnations(subjects[0].ID(), subjects[0].incarnationID, current.ID(), current.incarnationID)
			})
		}
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
