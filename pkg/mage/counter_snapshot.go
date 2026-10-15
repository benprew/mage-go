package mage

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CounterSnapshotBinding keeps each counter kind in the current resolution frame.
type CounterSnapshotBinding struct {
	counts [core.NumCounters]*NumberBinding
}

// BindCounterSnapshot declares a separate saved count for each counter kind.
func BindCounterSnapshot(name string) *CounterSnapshotBinding {
	binding := &CounterSnapshotBinding{}
	for counter := range core.NumCounters {
		binding.counts[counter] = BindNumber(fmt.Sprintf("%s counter %d", name, counter))
	}
	return binding
}

// SnapshotCounters saves all counter counts from the exact permanent visit.
// A departed or phased permanent supplies its last-known information.
func SnapshotCounters(subject PermanentRef, result *CounterSnapshotBinding) ResolvingEffect {
	var counts [core.NumCounters]*NumberBinding
	if result != nil {
		counts = result.counts
	}
	effects := make([]Effect, 0, core.NumCounters)
	for counter, binding := range counts {
		effects = append(effects, SnapshotNumber(CountersOnWithLKI(subject, core.CounterType(counter)), binding))
	}
	return Sequence(effects...)
}

type enterSavedCountersOption struct{ saved *CounterSnapshotBinding }

// EnterWithSavedCounters adds the saved counts as the card enters.
// The shared counter service applies replacements before entry observations.
func EnterWithSavedCounters(saved *CounterSnapshotBinding) BattlefieldEntryOption {
	return enterSavedCountersOption{saved}
}

func (enterSavedCountersOption) battlefieldEntryOption() {}
func (o enterSavedCountersOption) applyBattlefieldEntry(e *battlefieldEntryOptions) {
	e.savedCounters = o.saved
	e.savedCountersCount++
}

func (e battlefieldEntryOptions) captureCounters(ctx *EffectContext) battlefieldEntryOptions {
	if e.savedCounters != nil {
		for counter, binding := range e.savedCounters.counts {
			e.counterAmounts[counter] = FromNumber(binding).resolveValue(ctx)
		}
	}
	return e
}
