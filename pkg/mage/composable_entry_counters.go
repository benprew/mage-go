package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// EntryCounterValue reads a value while one permanent enters the battlefield.
type EntryCounterValue interface {
	entryCounterValue()
	entryCounterAmount(int) int
	inspectDefinition() definitionNode
}

type entryCastColorsValue struct{}

// ColorsSpentToCastEnteringPermanent reads the entering spell's own payment.
func ColorsSpentToCastEnteringPermanent() EntryCounterValue { return entryCastColorsValue{} }

func (entryCastColorsValue) entryCounterValue()           {}
func (entryCastColorsValue) entryCounterAmount(n int) int { return n }
func (entryCastColorsValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-counter-value", Text: "colors spent to cast entering permanent"}
}

type entryCountersAbility struct {
	BaseAbility
	counter core.CounterType
	amount  EntryCounterValue
}

// EntersWithCounters adds counters before the permanent enters the battlefield.
func EntersWithCounters(counter core.CounterType, amount EntryCounterValue) Ability {
	return &entryCountersAbility{
		BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic},
		counter:     counter,
		amount:      amount,
	}
}

func (a *entryCountersAbility) inspectDefinition() definitionNode {
	n := definitionNode{Role: "entry-replacement", Text: "enters with counters", children: []definitionEdge{{"amount", a.amount}}}
	if a.counter < 0 || a.counter >= core.NumCounters {
		n.problem = "invalid counter type"
	} else if nilDefinition(a.amount) {
		n.problem = "supply an entry amount"
	}
	return n
}
