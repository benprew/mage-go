package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithActivationManaXFromSourceCounters defines X before activation payment.
// Resolving instructions must read counters separately when their values can change.
func WithActivationManaXFromSourceCounters(counter core.CounterType) ActionOption {
	return func(a *ActionDefinition) {
		a.activationManaX = &activationManaX{counter: counter}
		a.activationManaXOptions++
	}
}

type activationManaX struct{ counter core.CounterType }

func (v *activationManaX) value(g *Game, sourceID uuid.UUID) (int, bool) {
	p := g.FindPermanent(sourceID)
	if p == nil || p.PhasedOut || v.counter < 0 || v.counter >= core.NumCounters {
		return 0, false
	}
	return int(p.Counters[v.counter]), true
}

func (v *activationManaX) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-mana-x", battlefieldSource: true}
	if v.counter < 0 || v.counter >= core.NumCounters {
		n.problem = "defined mana X requires a valid counter kind"
	}
	return n
}
