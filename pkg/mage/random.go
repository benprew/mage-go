package mage

import (
	"math/rand"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

var colors = [...]Color{White, Blue, Black, Red, Green}

// Random-card primitives. These cover effects that pick a card uniformly at
// random from a zone (Goblin Lore's "discard three cards at random",
// Charmbreaker Devils' "instant or sorcery card chosen at random from your
// graveyard", Ghoulraiser's "Zombie card at random").

// RandIntn returns a random integer in [0, n). For n <= 0 it returns 0 and
// does not consume a scripted result. SetRandomResults can supply deterministic
// raw values for tests; each value is normalized into range with mathematical
// modulo, so negative scripted values wrap from the end of the range. Once the
// scripted values are exhausted, RandIntn uses the process random source.
func (g *Game) RandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	if len(g.randomResults) == 0 {
		return rand.Intn(n)
	}
	result := g.randomResults[0]
	g.randomResults = g.randomResults[1:]
	result %= n
	if result < 0 {
		result += n
	}
	return result
}

// RandomPlayer returns one player chosen uniformly at random using the game's
// deterministic random source. It returns nil when the game has no players.
func (g *Game) RandomPlayer() Player {
	if len(g.players) == 0 {
		return nil
	}
	return g.players[g.RandIntn(len(g.players))]
}

// RandomPermanent returns one battlefield permanent chosen uniformly at random
// from those matching filter. A zero-value filter matches every permanent. It
// returns nil without consuming randomness when no permanent matches.
func (g *Game) RandomPermanent(filter PermanentFilter) *Permanent {
	var candidates []*Permanent
	for _, permanent := range g.AllBattlefield() {
		if filter.Match(permanent, g) {
			candidates = append(candidates, permanent)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[g.RandIntn(len(candidates))]
}

// RandomDamageTarget returns one creature or player chosen uniformly at random
// from all legal object kinds represented by "any target" in this engine. It
// returns uuid.Nil without consuming randomness when there are no candidates.
func (g *Game) RandomDamageTarget() uuid.UUID {
	var candidates []uuid.UUID
	for _, permanent := range g.AllBattlefield() {
		if IsCreature.Match(permanent, g) {
			candidates = append(candidates, permanent.ID())
		}
	}
	for _, player := range g.AllPlayers() {
		candidates = append(candidates, player.PlayerID())
	}
	if len(candidates) == 0 {
		return uuid.Nil
	}
	return candidates[g.RandIntn(len(candidates))]
}

// RandomColor returns one of Magic's five colors chosen uniformly at random
// using the game's deterministic random source.
func (g *Game) RandomColor() Color {
	return colors[g.RandIntn(len(colors))]
}

// RandomSpellOrPermanent returns the source ID of one spell on the stack or
// the ID of one battlefield permanent, chosen uniformly from the combined
// candidates. Abilities on the stack are excluded. It returns uuid.Nil without
// consuming randomness when there are no candidates.
func (g *Game) RandomSpellOrPermanent() uuid.UUID {
	var candidates []uuid.UUID
	for _, permanent := range g.AllBattlefield() {
		candidates = append(candidates, permanent.ID())
	}
	for _, object := range g.StackObjects() {
		if object.Card != nil && !object.IsAbility {
			candidates = append(candidates, object.SourceID)
		}
	}
	if len(candidates) == 0 {
		return uuid.Nil
	}
	return candidates[g.RandIntn(len(candidates))]
}

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
		idx := g.RandIntn(len(hand))
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
	return candidates[g.RandIntn(len(candidates))]
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
	return candidates[g.RandIntn(len(candidates))]
}
