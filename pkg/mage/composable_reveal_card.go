package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PublicCardReveal records one card shown to all players.
type PublicCardReveal struct {
	CardID uuid.UUID
	Owner  uuid.UUID
	Name   string
	Types  []core.CardType
	Card   Card
}

func clonePublicReveals(in []PublicCardReveal) []PublicCardReveal {
	if len(in) == 0 {
		return nil
	}
	out := make([]PublicCardReveal, len(in))
	for i, reveal := range in {
		out[i] = reveal
		out[i].Types = slices.Clone(reveal.Types)
		if reveal.Card != nil {
			out[i].Card = reveal.Card.Copy()
			out[i].Card.SetID(reveal.CardID)
		}
	}
	return out
}

// PublicReveals returns public reveal history in reveal order.
func (g *Game) PublicReveals() []PublicCardReveal {
	return clonePublicReveals(g.publicReveals)
}

func (g *Game) recordPublicReveal(card Card) {
	if card == nil {
		return
	}
	copyCard := card.Copy()
	copyCard.SetID(card.ID())
	g.publicReveals = append(g.publicReveals, PublicCardReveal{
		CardID: card.ID(), Owner: card.Owner(), Name: card.Name(), Types: slices.Clone(card.Types()), Card: copyCard,
	})
}

type revealCardEffect struct{ card CardRef }

// RevealCard shows the referenced card to all players if it is still present.
func RevealCard(card CardRef) ResolvingEffect { return &revealCardEffect{card: card} }

func (*revealCardEffect) Text() string                 { return "reveal the referenced card" }
func (*revealCardEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *revealCardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
}
func (e *revealCardEffect) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if ok {
		ctx.Game.recordPublicReveal(card)
	}
	return nil
}
