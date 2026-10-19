package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type auraRestrictionRule struct {
	subject   PermanentRef
	exception PermanentRef
	forbidAll bool
}

// CantBeEnchanted prevents all Auras from enchanting the subject.
func CantBeEnchanted(subject PermanentRef) StaticRule {
	return auraRestrictionRule{subject: subject, forbidAll: true}
}

// CantBeEnchantedExceptBy permits only the specified Aura to enchant the subject.
func CantBeEnchantedExceptBy(subject, exception PermanentRef) StaticRule {
	return auraRestrictionRule{subject: subject, exception: exception}
}

func (auraRestrictionRule) staticRule() {}

func (r auraRestrictionRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "only the specified Aura can enchant the subject", battlefieldSource: true}
	if r.forbidAll {
		n.Text = "no Aura can enchant the subject"
		if nilDefinition(r.subject) {
			n.problem = "Aura restriction requires a subject reference"
		} else {
			n.children = []definitionEdge{{"subject", r.subject}}
		}
	} else if nilDefinition(r.subject) || nilDefinition(r.exception) {
		n.problem = "Aura restriction requires subject and exception references"
	} else {
		n.children = []definitionEdge{{"subject", r.subject}, {"exception", r.exception}}
	}
	return n
}

func (r auraRestrictionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&auraRestriction{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		subject:              r.subject, exception: r.exception, forbidAll: r.forbidAll,
	}}
}

type auraRestriction struct {
	*funcContinuousEffect
	subject   PermanentRef
	exception PermanentRef
	forbidAll bool
}

func (e *auraRestriction) inspectDefinition() definitionNode {
	return auraRestrictionRule{subject: e.subject, exception: e.exception, forbidAll: e.forbidAll}.inspectDefinition()
}

func (e *auraRestriction) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

func (g *Game) restrictedAuraAttachment(host *Permanent, aura Card) bool {
	if g.preventsNewAuraAttachment(host, aura) {
		return true
	}
	if aura == nil || !aura.HasSubType("Aura") {
		return false
	}
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*auraRestriction)
		if !ok {
			continue
		}
		source := g.FindPermanent(rule.SourceID())
		if source == nil {
			continue
		}
		ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID}
		subject, _ := rule.subject.resolvePermanent(ctx)
		if subject == nil || subject.ID() != host.ID() || subject.incarnationID != host.incarnationID {
			continue
		}
		if rule.forbidAll {
			return true
		}
		exception, _ := rule.exception.resolvePermanent(ctx)
		candidate := g.FindPermanent(aura.ID())
		if exception == nil || candidate == nil || exception.ID() != candidate.ID() || exception.incarnationID != candidate.incarnationID {
			return true
		}
	}
	return false
}
