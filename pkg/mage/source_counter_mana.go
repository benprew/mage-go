package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type sourceCounterManaPlan struct {
	color   core.Color
	counter core.CounterType
}

// NewSourceCounterManaAbility adds one mana of the given type per source counter.
// Payment queries and activation read the same live count. Counters are not spent.
func NewSourceCounterManaAbility(color core.Color, counter core.CounterType) *ManaAbility {
	ability := NewManaAbility(color)
	ability.Productions = nil
	ability.sourceCounters = &sourceCounterManaPlan{color: color, counter: counter}
	return ability
}

func (p *sourceCounterManaPlan) productions(g GameReader, sourceID uuid.UUID) []ManaProduction {
	if g == nil || p.counter < 0 || p.counter >= core.NumCounters {
		return nil
	}
	source := g.FindPermanent(sourceID)
	if source == nil || source.PhasedOut {
		return nil
	}
	count := source.Counters[p.counter]
	if count <= 0 {
		return []ManaProduction{}
	}
	return []ManaProduction{{Color: p.color, Amount: count}}
}

func (p *sourceCounterManaPlan) inspectDefinition() definitionNode {
	node := definitionNode{Role: "source-counter-mana"}
	if p.color < core.Colorless || p.color >= core.AnyColor {
		node.problem = "supply one concrete mana type"
	}
	if p.counter < 0 || p.counter >= core.NumCounters {
		node.problem = "supply a permanent counter type"
	}
	return node
}
