package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type retainedProtectionRule struct {
	subjects  PermanentSet
	condition StateCondition
	control   bool
}

// PreventNewAuraAttachmentsIf permits existing Auras while the condition holds.
func PreventNewAuraAttachmentsIf(subjects PermanentSet, condition StateCondition) StaticRule {
	return retainedProtectionRule{subjects: subjects, condition: condition}
}

// PreventControlGainsIf preserves current control while the condition holds.
func PreventControlGainsIf(subjects PermanentSet, condition StateCondition) StaticRule {
	return retainedProtectionRule{subjects: subjects, condition: condition, control: true}
}

func (retainedProtectionRule) staticRule() {}
func (r retainedProtectionRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "prevent new Aura attachments", battlefieldSource: true}
	if r.control {
		n.Text = "prevent control gains"
	}
	if nilDefinition(r.subjects) || nilDefinition(r.condition) {
		n.problem = "protection requires subjects and a condition"
	} else {
		n.children = []definitionEdge{{"subjects", r.subjects}, {"condition", r.condition}}
	}
	return n
}

func (r retainedProtectionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&retainedProtectionEffect{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		rule:                 r,
	}}
}

type retainedProtectionEffect struct {
	*funcContinuousEffect
	rule retainedProtectionRule
}

func (e *retainedProtectionEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *retainedProtectionEffect) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

func (g *Game) retainedProtectionApplies(subject *Permanent, control bool) bool {
	for _, effect := range g.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		protection, ok := inner.(*retainedProtectionEffect)
		if !ok || protection.rule.control != control {
			continue
		}
		if !effect.IsActive(g) {
			continue
		}
		source := g.FindPermanent(effect.SourceID())
		if source == nil {
			continue
		}
		state := StateContext{SourceID: source.ID(), ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID}
		if !protection.rule.condition.checkState(g, state) {
			continue
		}
		ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
		for _, member := range protection.rule.subjects.resolvePermanents(ctx) {
			if member.ID() == subject.ID() && member.incarnationID == subject.incarnationID {
				return true
			}
		}
	}
	return false
}

func (g *Game) preventsNewAuraAttachment(host *Permanent, aura Card) bool {
	if aura == nil || !aura.HasSubType("Aura") {
		return false
	}
	if attached := g.FindPermanent(aura.ID()); attached != nil && attached.AttachedTo == host.ID() && attached.attachedIncarnation == host.incarnationID {
		return false
	}
	return g.retainedProtectionApplies(host, false)
}
