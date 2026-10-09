package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RevealHandCardInto saves the exact hand visit and facts of a public reveal.
// An absent card clears the result. A returned card cannot replace its visit.
func RevealHandCardInto(card CardRef, result *CardBinding) ResolvingEffect {
	return &savedHandRevealEffect{card: card, result: result}
}

type savedHandRevealEffect struct {
	card   CardRef
	result *CardBinding
}

func (*savedHandRevealEffect) Text() string                 { return "reveal the saved hand card" }
func (*savedHandRevealEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *savedHandRevealEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", bindingWrite: e.result, children: []definitionEdge{{"card", e.card}, {"result", e.result}}}
}
func (e *savedHandRevealEffect) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	delete(ctx.Game.resolution.cardBindings, e.result)
	if !ok {
		return nil
	}
	visit := ctx.Game.handVisit(card.ID())
	if visit == uuid.Nil {
		return nil
	}
	if ctx.Game.resolution.cardBindings == nil {
		ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
	}
	ctx.Game.resolution.cardBindings[e.result] = capturedCard{id: card.ID(), visit: visit, zone: core.ZoneHand, revealed: &revealedCardFacts{name: card.Name(), types: slices.Clone(card.Types())}}
	ctx.Game.recordPublicReveal(card)
	return nil
}
