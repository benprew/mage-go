package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

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

// CountCountersAmong reads counters on current members of a collection.
// Objects that have left the battlefield cannot supply counters to this total.
func CountCountersAmong(subjects PermanentSet, counter core.CounterType) ResolutionValue {
	return permanentCounterTotal{subjects: subjects, counter: counter}
}

type permanentCounterTotal struct {
	subjects PermanentSet
	counter  core.CounterType
}

func (permanentCounterTotal) Text() string { return "total counters among permanents" }

func (v permanentCounterTotal) resolveValue(ctx *EffectContext) int {
	if nilDefinition(v.subjects) || v.counter < 0 || v.counter >= core.NumCounters {
		return 0
	}
	total := 0
	seen := map[uuid.UUID]bool{}
	for _, member := range v.subjects.resolvePermanents(ctx) {
		if member == nil {
			continue
		}
		current := ctx.Game.FindPermanent(member.ID())
		if current == nil || current.PhasedOut || current.incarnationID != member.incarnationID || seen[current.incarnationID] {
			continue
		}
		seen[current.incarnationID] = true
		total += int(current.Counters[v.counter])
	}
	return total
}

func (v permanentCounterTotal) inspectDefinition() definitionNode {
	n := definitionNode{Role: "counter-total", Text: v.Text(), children: []definitionEdge{{"subjects", v.subjects}}}
	if v.counter < 0 || v.counter >= core.NumCounters {
		n.problem = "counter total requires a valid counter kind"
	}
	return n
}
