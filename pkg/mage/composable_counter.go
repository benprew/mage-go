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

type counterEachEffect struct {
	subjects PermanentSet
	counter  core.CounterType
	amount   ResolutionValue
}

// AddCountersEach places counters on the selected set in one instruction.
func AddCountersEach(subjects PermanentSet, counter core.CounterType, amount ResolutionValue) ResolvingEffect {
	return &counterEachEffect{subjects: subjects, counter: counter, amount: amount}
}

func (*counterEachEffect) Text() string { return "add counters to each permanent" }
func (e *counterEachEffect) Properties() EffectProperties {
	properties := EffectProperties{Mass: true}
	if amount, ok := e.amount.(fixedValue); ok {
		properties.Outcome = AddCounters(e.counter, amount).Properties().Outcome
	}
	return properties
}
func (e *counterEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "counter-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
	if e.counter < 0 || e.counter >= core.NumCounters {
		n.problem = "supply a valid counter type"
	}
	return n
}
func (e *counterEachEffect) Apply(ctx *EffectContext) error {
	selected := e.subjects.resolvePermanents(ctx)
	if len(selected) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	ctx.Game.AddCountersTogether(selected, e.counter, amount, ctx.SourceID)
	return nil
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
