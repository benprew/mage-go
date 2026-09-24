package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type removeCountersSubjectEffect struct {
	subject PermanentRef
	counter core.CounterType
	amount  ResolutionValue
}

// RemoveCountersFrom removes up to the specified number of counters from a subject.
func RemoveCountersFrom(subject PermanentRef, counter core.CounterType, amount ResolutionValue) ResolvingEffect {
	return &removeCountersSubjectEffect{subject, counter, amount}
}
func (e *removeCountersSubjectEffect) Text() string {
	return "remove " + e.counter.String() + " counters from " + InspectDefinition(e.subject).Text
}
func (*removeCountersSubjectEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *removeCountersSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *removeCountersSubjectEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	amount := min(e.amount.resolveValue(ctx), int(p.Counters[e.counter]))
	if amount <= 0 {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{p.ID()}
	return RemoveCounters(e.counter, amount).Targeting(ToTarget()).Apply(&child)
}
