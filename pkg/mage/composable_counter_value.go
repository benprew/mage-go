package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type counterResolutionValue struct {
	subject PermanentRef
	counter core.CounterType
}

// CountersOn reads the current count of one counter kind on a permanent.
// An absent permanent supplies zero. This value does not use last-known data.
func CountersOn(subject PermanentRef, counter core.CounterType) ResolutionValue {
	if counter < 0 || counter >= core.NumCounters {
		panic("CountersOn requires a valid counter kind")
	}
	return counterResolutionValue{subject: subject, counter: counter}
}

func (v counterResolutionValue) Text() string {
	return v.counter.String() + " counters on subject"
}

func (v counterResolutionValue) resolveValue(ctx *EffectContext) int {
	p, ok := v.subject.resolvePermanent(ctx)
	if !ok {
		return 0
	}
	return int(p.Counters[v.counter])
}

func (v counterResolutionValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "counter-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}
