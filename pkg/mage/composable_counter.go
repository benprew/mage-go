package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type counterSubjectEffect struct {
	subject PermanentRef
	counter core.CounterType
	amount  ResolutionValue
}

func (e *counterSubjectEffect) Text() string {
	return "add counters to " + InspectDefinition(e.subject).Text
}
func (e *counterSubjectEffect) Properties() EffectProperties {
	if amount, ok := e.amount.(fixedValue); ok {
		return AddCounters(e.counter, amount).Properties()
	}
	return EffectProperties{}
}
func (e *counterSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *counterSubjectEffect) Apply(ctx *EffectContext) error {
	permanent, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{permanent.ID()}
	return AddCounters(e.counter, Fixed(amount)).Apply(&child)
}
