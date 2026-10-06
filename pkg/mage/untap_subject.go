package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type subjectUntapRule struct {
	subject   PermanentRef
	condition StateCondition
}

// PreventUntapDuringStep checks the subject and condition before simultaneous untap.
// The restriction stays separate from the subject's abilities.
func PreventUntapDuringStep(subject PermanentRef, condition StateCondition) StaticRule {
	return subjectUntapRule{subject: subject, condition: condition}
}

func (subjectUntapRule) staticRule() {}
func (r subjectUntapRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-rule", Text: "subject cannot untap during untap step", battlefieldSource: true,
		children: []definitionEdge{{"subject", r.subject}, {"condition", r.condition}}}
}
func (r subjectUntapRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&subjectUntapRestriction{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		rule:                 r,
	}}
}

type subjectUntapRestriction struct {
	*funcContinuousEffect
	rule subjectUntapRule
}

func (e *subjectUntapRestriction) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *subjectUntapRestriction) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

func (e *subjectUntapRestriction) subjectAtDecision(g *Game) *Permanent {
	source := g.FindPermanent(e.SourceID())
	if source == nil || nilDefinition(e.rule.subject) || nilDefinition(e.rule.condition) {
		return nil
	}
	state := StateContext{SourceID: source.ID(), ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID}
	if !e.rule.condition.checkState(g, state) {
		return nil
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID}
	subject, ok := e.rule.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	return subject
}
