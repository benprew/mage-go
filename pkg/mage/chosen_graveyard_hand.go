package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ReturnChosenGraveyardCardsToHand makes all choices before the cards move.
// Each choice keeps the saved graveyard visit and returns to its owner's hand.
func ReturnChosenGraveyardCardsToHand(graveyard PlayerRef, amount ResolutionValue, filter CardFilter) ResolvingEffect {
	return &chosenGraveyardHand{graveyard, amount, filter}
}

type chosenGraveyardHand struct {
	graveyard PlayerRef
	amount    ResolutionValue
	filter    CardFilter
}

func (*chosenGraveyardHand) Text() string {
	return "return chosen graveyard cards to their owners' hands"
}
func (*chosenGraveyardHand) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *chosenGraveyardHand) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"graveyard", e.graveyard}, {"amount", e.amount}}}
	if e.filter.IsZero() {
		n.problem = "supply a graveyard card filter"
	}
	return n
}
func (e *chosenGraveyardHand) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.graveyard) || resolutionValueReadsSourcePermanent(e.amount)
}
func (e *chosenGraveyardHand) Apply(ctx *EffectContext) error {
	id, ok := e.graveyard.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player, chooser := ctx.Game.GetPlayer(id), ctx.Game.GetPlayer(ctx.Controller)
	if player == nil || chooser == nil {
		return nil
	}
	amount := max(0, e.amount.resolveValue(ctx))
	if amount == 0 {
		return nil
	}
	var candidates []Card
	visits := make(map[uuid.UUID]uuid.UUID)
	for _, card := range player.Graveyard() {
		if e.filter.Match(card) && visits[card.ID()] == uuid.Nil {
			candidates = append(candidates, card)
			visits[card.ID()] = ctx.Game.graveyardVisit(card.ID())
		}
	}
	count := min(amount, len(candidates))
	chosen := make([]uuid.UUID, 0, count)
	for range count {
		index := 0
		if len(candidates) > 1 {
			selection := chooser.ChooseCardFromGraveyard(append([]Card(nil), candidates...), "Choose a card to return to hand", ctx.Game)
			if selection != nil {
				for i, card := range candidates {
					if card.ID() == selection.ID() {
						index = i
						break
					}
				}
			}
		}
		chosen = append(chosen, candidates[index].ID())
		candidates = append(candidates[:index], candidates[index+1:]...)
	}
	var current []uuid.UUID
	for _, cardID := range chosen {
		if visits[cardID] != uuid.Nil && ctx.Game.graveyardVisit(cardID) == visits[cardID] {
			current = append(current, cardID)
		}
	}
	ctx.Game.returnGraveyardCardsToOwnersHands(id, current)
	return nil
}

func (g *Game) returnGraveyardCardsToOwnersHands(playerID uuid.UUID, ids []uuid.UUID) {
	player := g.GetPlayer(playerID)
	if player == nil {
		return
	}
	var selected []uuid.UUID
	for _, id := range ids {
		for _, card := range player.Graveyard() {
			if card.ID() == id && g.GetPlayer(card.Owner()) != nil {
				selected = append(selected, id)
				break
			}
		}
	}
	g.moveCardsFromGraveyard(playerID, selected, core.ZoneHand, func(card Card) { g.GetPlayer(card.Owner()).AddToHand(card) })
}
