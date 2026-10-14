package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// ExiledCardHasCounter checks one present, face-up exile visit for a counter.
func ExiledCardHasCounter(card CardRef, counter core.CounterType) StateCondition {
	return exileCounterCondition{card: card, counter: counter}
}

// ExiledCardHasNoCounters checks one present, face-up exile visit for no counters of a type.
func ExiledCardHasNoCounters(card CardRef, counter core.CounterType) StateCondition {
	return exileCounterCondition{card: card, counter: counter, none: true}
}

type exileCounterCondition struct {
	card    CardRef
	counter core.CounterType
	none    bool
}

func (exileCounterCondition) stateCondition() {}
func (c exileCounterCondition) checkState(reader GameReader, state StateContext) bool {
	ctx := cardConditionContext(reader, state)
	if ctx == nil || c.counter < 0 || c.counter >= core.NumCounters {
		return false
	}
	ref, ok := c.card.(savedCardRef)
	if !ok {
		return false
	}
	identity, ok := state.cards[ref.binding]
	if !ok || identity.zone != core.ZoneExile {
		return false
	}
	card, ok := ref.resolveCard(ctx)
	if !ok {
		return false
	}
	object := ctx.Game.FindExiledCard(card.ID())
	if object == nil || object.FaceDown {
		return false
	}
	if c.none {
		return object.Counters[c.counter] == 0
	}
	return object.Counters[c.counter] > 0
}

func (c exileCounterCondition) inspectDefinition() definitionNode {
	text := "saved exiled card has a counter"
	if c.none {
		text = "saved exiled card has no counters of a type"
	}
	n := definitionNode{Role: "state-condition", Text: text, children: []definitionEdge{{"card", c.card}}}
	if _, ok := c.card.(savedCardRef); !ok {
		n.problem = "exile counter condition requires a saved card reference"
	}
	if c.counter < 0 || c.counter >= core.NumCounters {
		n.problem = "exile counter condition requires a valid counter type"
	}
	return n
}
