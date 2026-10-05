package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type exileGraveyardCardEffect struct{ card CardRef }

// ExileGraveyardCard uses the saved visit so a returned card is not moved.
func ExileGraveyardCard(card CardRef) ResolvingEffect {
	return &exileGraveyardCardEffect{card: card}
}

func (*exileGraveyardCardEffect) Text() string { return "exile graveyard card" }

func (*exileGraveyardCardEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *exileGraveyardCardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
}

func (e *exileGraveyardCardEffect) Apply(ctx *EffectContext) error {
	if e.card == nil {
		return nil
	}
	card, ok := e.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	moved, _, ok := ctx.Game.MoveFromAnyGraveyard(card.ID(), core.ZoneExile)
	if ok {
		ctx.Game.ExileCard(moved, ctx.SourceID)
	}
	return nil
}
