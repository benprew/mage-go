package mage

// inspectGlobalTargetRestrictions reads only known native restriction contracts.
// Opaque activity, subjects and predicates cannot establish target legality.
func (g *Game) inspectGlobalTargetRestrictions(p *Permanent, obj *StackObject, source Card) (restricted, known bool) {
	for _, effect := range g.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		var subject PermanentRef
		var filter CardFilter
		var base *funcContinuousEffect
		var condition StateCondition
		conditional := false
		switch rule := inner.(type) {
		case *spellTargetRestriction:
			if obj.IsAbility {
				continue
			}
			if rule == nil {
				return false, false
			}
			subject, filter, base, conditional = rule.subject, rule.filter, rule.funcContinuousEffect, rule.conditional
			condition = rule.condition
		case *abilityTargetRestriction:
			if !obj.IsAbility {
				continue
			}
			if rule == nil {
				return false, false
			}
			subject, filter, base = rule.subject, rule.filter, rule.funcContinuousEffect
		default:
			continue
		}
		if base == nil || base.active != nil {
			return false, false
		}
		if !effect.IsActive(g) {
			continue
		}
		ruleSource := g.FindPermanent(base.SourceID())
		if ruleSource == nil {
			continue
		}
		switch subject.(type) {
		case sourceRef, attachedPermanentRef:
		default:
			return false, false
		}
		ctx := &EffectContext{Game: g, SourceID: ruleSource.ID(), Controller: ruleSource.ControllerID(), sourceCaptured: true, sourceIncarnation: ruleSource.incarnationID}
		target, _ := subject.resolvePermanent(ctx)
		if target == nil || target.ID() != p.ID() || target.incarnationID != p.incarnationID {
			continue
		}
		if filter.characteristics == nil {
			return false, false
		}
		if !filter.characteristics.matches(source) {
			continue
		}
		if conditional {
			if !inspectableTargetCondition(condition) {
				return false, false
			}
			if !condition.checkState(g, StateContext{SourceID: ruleSource.ID(), ControllerID: ruleSource.ControllerID(), SourceIncarnation: ruleSource.incarnationID}) {
				continue
			}
		}
		return true, true
	}
	return false, true
}
