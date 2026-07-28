package mage

import (
	"math/rand"

	"github.com/google/uuid"
)

// Random-card primitives. These cover effects that pick a card uniformly at
// random from a zone (Goblin Lore's "discard three cards at random",
// Charmbreaker Devils' "instant or sorcery card chosen at random from your
// graveyard", Ghoulraiser's "Zombie card at random").
//
// Random selection uses math/rand directly (the same RNG already used by
// ShuffleLibrary, DiscardRandomCost, Chaos Orb, and Aladdin's Lamp). When
// the engine grows a seedable RNG, every consumer migrates together.

// DiscardAtRandom forces `p` to discard up to n cards from their hand,
// chosen uniformly at random. Returns the discarded cards in the order they
// were discarded. Stops early if the hand empties. Each discard fires
// EvtDiscard via Game.PlayerDiscard, so triggers like Liliana's Caress see
// every card individually.
//
// Unlike the existing DiscardRandom Effect (which targets an opponent),
// this primitive operates on a specific player and is suitable for self-
// discard ("you discard ... at random", as on Goblin Lore).
func (g *Game) DiscardAtRandom(p Player, n int) []Card {
	if p == nil || n <= 0 {
		return nil
	}
	var discarded []Card
	for range n {
		hand := p.Hand()
		if len(hand) == 0 {
			break
		}
		idx := rand.Intn(len(hand))
		card := hand[idx]
		if c, ok := g.PlayerDiscardByEffect(p, card.ID(), uuid.Nil); ok {
			discarded = append(discarded, c)
		} else {
			break
		}
	}
	return discarded
}

// RandomCardFromGraveyard returns one card chosen uniformly at random from
// `p`'s graveyard that matches `filter`. Returns nil if no card matches or
// the graveyard is empty. The card is NOT removed from the graveyard;
// callers that need to move it use p.RemoveFromGraveyard(card.ID()).
//
// Use a zero-value CardFilter (or one whose Match always returns true) for
// "any card" — Charmbreaker Devils and similar specify a card type, while
// Ghoulraiser-style filters narrow further by subtype.
func (g *Game) RandomCardFromGraveyard(p Player, filter CardFilter) Card {
	if p == nil {
		return nil
	}
	var candidates []Card
	for _, c := range p.Graveyard() {
		if filter.Match(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rand.Intn(len(candidates))]
}

// RandomCardFromHand returns one card chosen uniformly at random from `p`'s
// hand that matches `filter`. Returns nil if no card matches. The card is
// NOT removed from the hand; callers move it themselves. Useful for "reveal
// a card at random from your hand" patterns and AI heuristics that need a
// random hand card.
func (g *Game) RandomCardFromHand(p Player, filter CardFilter) Card {
	if p == nil {
		return nil
	}
	var candidates []Card
	for _, c := range p.Hand() {
		if filter.Match(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rand.Intn(len(candidates))]
}
