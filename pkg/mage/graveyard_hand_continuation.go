package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type graveyardHandContinuation struct {
	card            CardRef
	moved, notMoved ResolvingEffect
}

// ReturnCardToHandOrElse selects a branch after the exact graveyard card moves.
// Each call keeps its own result, including calls in a branch.
func ReturnCardToHandOrElse(card CardRef, moved, notMoved ResolvingEffect) ResolvingEffect {
	return &graveyardHandContinuation{card: card, moved: moved, notMoved: notMoved}
}
func (*graveyardHandContinuation) Text() string {
	return "return graveyard card to hand and select a branch"
}
func (e *graveyardHandContinuation) Properties() EffectProperties {
	props := EffectProperties{Outcome: OutcomeBenefit}
	for _, body := range []ResolvingEffect{e.moved, e.notMoved} {
		if !nilDefinition(body) {
			mergeEffectProperties(&props, body.Properties())
		}
	}
	return props
}
func (e *graveyardHandContinuation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
	if !nilDefinition(e.moved) {
		n.children = append(n.children, definitionEdge{"moved", e.moved})
	}
	if !nilDefinition(e.notMoved) {
		n.children = append(n.children, definitionEdge{"not-moved", e.notMoved})
	}
	if nilDefinition(e.moved) && nilDefinition(e.notMoved) {
		n.problem = "supply at least one zone move branch"
	}
	return n
}
func (e *graveyardHandContinuation) Apply(ctx *EffectContext) error {
	body := e.notMoved
	if returnGraveyardCardToHand(ctx, e.card) {
		body = e.moved
	}
	if !nilDefinition(body) {
		return body.Apply(ctx)
	}
	return nil
}
func (e *graveyardHandContinuation) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.moved) || effectReadsSourcePermanent(e.notMoved)
}

func returnGraveyardCardToHand(ctx *EffectContext, ref CardRef) bool {
	if nilDefinition(ref) {
		return false
	}
	card, ok := ref.resolveCard(ctx)
	if !ok || card.IsToken() {
		return false
	}
	owner := ctx.Game.GetPlayer(card.Owner())
	if owner == nil {
		return false
	}
	moved, _, ok := ctx.Game.MoveFromAnyGraveyard(card.ID(), core.ZoneHand)
	if !ok {
		return false
	}
	owner.AddToHand(moved)
	return true
}
