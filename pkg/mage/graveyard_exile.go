package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

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
	g := ctx.Game
	visit := g.graveyardVisit(card.ID())
	if visit == uuid.Nil || card.IsToken() {
		return nil
	}
	proposal := &CardZoneMoveAction{source: ctx.SourceID, card: card.ID(), visit: visit, from: core.ZoneGraveyard, destination: core.ZoneExile, owner: card.Owner(), exileSource: ctx.SourceID}
	if result, ok := g.effects.ApplyReplacements(proposal, g).(*CardZoneMoveAction); ok && result != nil {
		g.performCardZoneMove(result)
	}
	return nil
}
