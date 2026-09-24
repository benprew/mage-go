package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type sourceCounterCondition struct {
	counter core.CounterType
	minimum int
}

// SourceHasCounters checks for at least minimum counters on the source.
func SourceHasCounters(counter core.CounterType, minimum int) StateCondition {
	return sourceCounterCondition{counter, minimum}
}
func (sourceCounterCondition) stateCondition() {}
func (c sourceCounterCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	return p != nil && !p.PhasedOut && int(p.Counters[c.counter]) >= c.minimum
}
func (c sourceCounterCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: fmt.Sprintf("source has at least %d %s counters", c.minimum, c.counter), battlefieldSource: true}
	if c.minimum < 0 {
		n.problem = "supply a nonnegative counter threshold"
	}
	return n
}

type removeAllCountersEffect struct {
	subject PermanentRef
	counter core.CounterType
}

// RemoveAllCounters removes all counters of one kind from the subject.
func RemoveAllCounters(subject PermanentRef, counter core.CounterType) ResolvingEffect {
	return &removeAllCountersEffect{subject, counter}
}
func (e *removeAllCountersEffect) Text() string {
	return "remove all " + e.counter.String() + " counters"
}
func (*removeAllCountersEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *removeAllCountersEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *removeAllCountersEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{p.ID()}
	return RemoveCounters(e.counter, int(p.Counters[e.counter])).Targeting(ToTarget()).Apply(&child)
}
