package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type sourceCounterValue struct{ counter core.CounterType }

// SourceCounters reads counters on the source incarnation at resolution.
// A departed source supplies its last count. An unknown source supplies zero.
func SourceCounters(counter core.CounterType) ResolutionValue {
	if counter < 0 || counter >= core.NumCounters {
		panic("SourceCounters requires a valid counter kind")
	}
	return sourceCounterValue{counter: counter}
}

func (v sourceCounterValue) Text() string {
	return v.counter.String() + " counters on source, with last-known information"
}

func (v sourceCounterValue) resolveValue(ctx *EffectContext) int {
	p := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID)
	if p != nil && (!ctx.sourceCaptured || p.incarnationID == ctx.sourceIncarnation) {
		return int(p.Counters[v.counter])
	}
	if ctx.sourceCaptured {
		return int(ctx.Game.zones.departedCounters[ctx.sourceIncarnation][v.counter])
	}
	return 0
}

func (v sourceCounterValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "source-counter-value", Text: v.Text(), battlefieldSource: true}
}
