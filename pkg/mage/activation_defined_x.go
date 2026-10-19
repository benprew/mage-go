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

// WithActivationManaXFromSpell fixes payment X from the sole spell target.
func WithActivationManaXFromSpell(spell *SpellTargetBinding, multiplier int) ActionOption {
	return func(a *ActionDefinition) {
		a.activationManaX = &activationManaX{spell: spell, multiplier: multiplier, fromSpell: true}
		a.activationManaXOptions++
	}
}

type activationManaX struct {
	counter    core.CounterType
	spell      *SpellTargetBinding
	multiplier int
	fromSpell  bool
}

func (v *activationManaX) value(g *Game, sourceID uuid.UUID, targets ...uuid.UUID) (int, bool) {
	if v.fromSpell {
		if len(targets) != 1 {
			return 0, false
		}
		obj := g.stack.FindByID(targets[0])
		if obj == nil || obj.IsAbility || obj.Card == nil || v.multiplier <= 0 {
			return 0, false
		}
		return v.multiplier * stackSpellManaValue(obj), true
	}
	p := g.FindPermanent(sourceID)
	if p == nil || p.PhasedOut || v.counter < 0 || v.counter >= core.NumCounters {
		return 0, false
	}
	return p.Counters[v.counter], true
}

func (v *activationManaX) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-mana-x", battlefieldSource: true}
	if v.fromSpell {
		n.children = []definitionEdge{{"spell", v.spell}}
		if v.multiplier <= 0 {
			n.problem = "supply a positive mana value multiplier"
		}
		return n
	}
	if v.counter < 0 || v.counter >= core.NumCounters {
		n.problem = "defined mana X requires a valid counter kind"
	}
	return n
}
