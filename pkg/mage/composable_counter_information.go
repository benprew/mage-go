package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// CountersOnWithLKI keeps counter information for the original permanent.
// A new incarnation cannot replace the saved information.
func CountersOnWithLKI(subject PermanentRef, counter core.CounterType) ResolutionValue {
	return permanentCounterInformation{subject: subject, counter: counter}
}

type permanentCounterInformation struct {
	subject PermanentRef
	counter core.CounterType
}

func (v permanentCounterInformation) Text() string {
	return "counter count with last-known information"
}

func (v permanentCounterInformation) resolveValue(ctx *EffectContext) int {
	p, lki := readPermanentInformation(v.subject, ctx)
	if lki != nil {
		p = lki.Snapshot
	}
	if p == nil || v.counter < 0 || v.counter >= core.NumCounters {
		return 0
	}
	return int(p.Counters[v.counter])
}

func (v permanentCounterInformation) inspectDefinition() definitionNode {
	n := permanentInformationNode(v.subject, "counter-value", v.Text())
	if v.counter < 0 || v.counter >= core.NumCounters {
		n.problem = "counter information requires a valid counter kind"
	}
	return n
}
