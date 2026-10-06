package mage

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// HasCounter matches a permanent with one or more counters of the selected kind.
func HasCounter(counter core.CounterType) PermanentFilter {
	return NewPermanentFilter("has "+counter.String()+" counter", func(p *Permanent, _ *Game) bool {
		return p != nil && p.Counters[counter] > 0
	})
}

// CounterCountEquals matches the current count without a narrow integer cast.
func CounterCountEquals(counter core.CounterType, count int) PermanentFilter {
	return NewPermanentFilter(fmt.Sprintf("has exactly %d %s counters", count, counter), func(p *Permanent, _ *Game) bool {
		return p != nil && int(p.Counters[counter]) == count
	})
}
