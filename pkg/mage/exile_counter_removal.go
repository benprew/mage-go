package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type exileCounterRemovalEffect struct {
	card    CardRef
	counter core.CounterType
}

// RemoveCounterFromExiledCard removes one counter from the saved exile visit.
func RemoveCounterFromExiledCard(card CardRef, counter core.CounterType) ResolvingEffect {
	return &exileCounterRemovalEffect{card: card, counter: counter}
}

func (*exileCounterRemovalEffect) Text() string                 { return "remove one counter from an exiled card" }
func (*exileCounterRemovalEffect) Properties() EffectProperties { return EffectProperties{} }

func (e *exileCounterRemovalEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
	if _, ok := e.card.(savedCardRef); !ok {
		n.problem = "exile counter removal requires a saved card reference"
	}
	if e.counter < 0 || e.counter >= core.NumCounters {
		n.problem = "exile counter removal requires a valid counter type"
	}
	return n
}

func (e *exileCounterRemovalEffect) Apply(ctx *EffectContext) error {
	ref, ok := e.card.(savedCardRef)
	if !ok || e.counter < 0 || e.counter >= core.NumCounters {
		return nil
	}
	identity, ok := ctx.Game.resolution.cardBindings[ref.binding]
	if !ok || identity.zone != core.ZoneExile {
		return nil
	}
	card, ok := ref.resolveCard(ctx)
	if !ok {
		return nil
	}
	object := ctx.Game.FindExiledCard(card.ID())
	if object == nil || object.FaceDown || object.Counters[e.counter] <= 0 {
		return nil
	}
	object.Counters[e.counter]--
	return nil
}
