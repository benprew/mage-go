package mage

import (
	"errors"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type randomHandRevealEffect struct {
	player PlayerRef
	result *CardBinding
}

// RevealRandomHandCardInto saves the exact hand visit of a public random reveal.
// An empty hand or absent player clears the result without using randomness.
func RevealRandomHandCardInto(player PlayerRef, result *CardBinding) ResolvingEffect {
	return &randomHandRevealEffect{player: player, result: result}
}

func (*randomHandRevealEffect) Text() string {
	return "reveal a random card from the selected player's hand"
}
func (*randomHandRevealEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *randomHandRevealEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", bindingWrite: e.result, children: []definitionEdge{{"player", e.player}, {"result", e.result}}}
}
func (e *randomHandRevealEffect) Apply(ctx *EffectContext) error {
	state := &ctx.Game.resolution
	delete(state.cardBindings, e.result)
	player := handOperationPlayer(e.player, ctx)
	card := ctx.Game.RandomCardFromHand(player, CardFilter{})
	if card == nil {
		return nil
	}
	visit := ctx.Game.handVisit(card.ID())
	if visit == uuid.Nil {
		return errors.New("random hand reveal requires an engine hand-visit identity")
	}
	if state.cardBindings == nil {
		state.cardBindings = make(map[*CardBinding]capturedCard)
	}
	state.cardBindings[e.result] = capturedCard{id: card.ID(), visit: visit, zone: core.ZoneHand, revealed: &revealedCardFacts{name: card.Name(), types: slices.Clone(card.Types())}}
	ctx.Game.recordPublicReveal(card)
	return nil
}
func (e *randomHandRevealEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

type discardCardFromHandEffect struct {
	player PlayerRef
	card   CardRef
}

// DiscardCardFromHandFor keeps the explicit player and the saved card visit.
// The shared discard service applies replacements and sends discard events.
func DiscardCardFromHandFor(player PlayerRef, card CardRef) ResolvingEffect {
	return &discardCardFromHandEffect{player: player, card: card}
}
func (*discardCardFromHandEffect) Text() string {
	return "discard the referenced card from the selected player's hand"
}
func (*discardCardFromHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardCardFromHandEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}, {"card", e.card}}}
}
func (e *discardCardFromHandEffect) Apply(ctx *EffectContext) error {
	player := handOperationPlayer(e.player, ctx)
	card, ok := e.card.resolveCard(ctx)
	if player == nil || !ok {
		return nil
	}
	for _, held := range player.Hand() {
		if held.ID() == card.ID() {
			ctx.Game.PlayerDiscardByEffect(player, card.ID(), ctx.SourceID)
			break
		}
	}
	return nil
}
func (e *discardCardFromHandEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
