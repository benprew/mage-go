package mage

import (
	"math/rand"

	"github.com/google/uuid"
)

// Reveal-and-pick primitives. These power cards that look at the top N cards
// of a library, optionally let a player choose one matching a filter, and
// place the rest somewhere (typically the bottom of the library in a random
// order). They also cover "reveal hand" patterns where one player inspects
// another's hand and chooses a card from it.
//
// The functions in this file do not move cards on their own beyond what each
// primitive's documented contract specifies. Callers are responsible for the
// downstream zone change (put into hand, onto battlefield, into graveyard,
// etc.) once they have the chosen Card.

// RevealTopN returns a copy of the top n cards of the player's library
// without modifying the library. If the library has fewer than n cards, it
// returns whatever is present (possibly an empty slice). The returned slice
// is a copy; callers may freely mutate it.
//
// "Look at" and "reveal" both use this primitive at the engine level — the
// distinction (private knowledge vs public knowledge) is a UI concern that
// the engine does not currently model.
func (g *Game) RevealTopN(p Player, n int) []Card {
	if p == nil || n <= 0 {
		return nil
	}
	lib := p.Library()
	if len(lib) == 0 {
		return nil
	}
	count := min(n, len(lib))
	out := make([]Card, count)
	copy(out, lib[:count])
	return out
}

// RemoveTopN removes and returns the top n cards of the player's library.
// If the library has fewer than n cards, it returns whatever is present.
// Used together with relocation primitives such as PutOnBottomInRandomOrder.
func (g *Game) RemoveTopN(p Player, n int) []Card {
	if p == nil || n <= 0 {
		return nil
	}
	lib := p.Library()
	if len(lib) == 0 {
		return nil
	}
	count := min(n, len(lib))
	taken := make([]Card, count)
	copy(taken, lib[:count])
	rest := make([]Card, len(lib)-count)
	copy(rest, lib[count:])
	p.SetLibrary(rest)
	return taken
}

// RevealAndPickFromTop looks at the top n cards of the player's library,
// asks the chooser to pick one whose card matches `filter`, and returns
// the chosen card plus the remaining (un-chosen) cards. The chooser is the
// player passed in `chooser` — for "you" patterns, pass the controller; for
// "opponent picks" patterns, pass the opponent.
//
// If no revealed card matches the filter, chosen is nil and rest contains
// every revealed card. If `mayDecline` is true and at least one card matches,
// the chooser may still decline (return nil) — in which case chosen is nil
// and rest contains every revealed card.
//
// This primitive does NOT mutate the library. The caller must remove the
// chosen card from the library and place the rest somewhere; the typical
// pattern is RemoveTopN followed by inserting the chosen card into the
// destination zone and bottoming the rest in random order.
func (g *Game) RevealAndPickFromTop(chooser Player, owner Player, n int, filter CardFilter, mayDecline bool, reason string) (chosen Card, revealed []Card) {
	revealed = g.RevealTopN(owner, n)
	if len(revealed) == 0 {
		return nil, revealed
	}
	var candidates []Card
	for _, c := range revealed {
		if filter.Match(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil, revealed
	}
	chosen = chooser.ChooseCardFromLibrary(candidates, reason, g)
	if chosen == nil && !mayDecline {
		chosen = candidates[0]
	}
	// If mayDecline is true, chooser may legitimately return nil to decline.
	return chosen, revealed
}

// PutOnBottomInRandomOrder places the given cards at the bottom of the
// player's library in a uniformly random order (CR 701.20-style "random
// order" used by Aladdin's Lamp, Commune with Dinosaurs, Muxus, etc.). The
// cards must NOT currently be in the library. The caller is responsible for
// having removed them first (typically via RemoveTopN).
func (g *Game) PutOnBottomInRandomOrder(p Player, cards []Card) {
	if p == nil || len(cards) == 0 {
		return
	}
	shuffled := make([]Card, len(cards))
	copy(shuffled, cards)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	lib := p.Library()
	newLib := make([]Card, 0, len(lib)+len(shuffled))
	newLib = append(newLib, lib...)
	newLib = append(newLib, shuffled...)
	p.SetLibrary(newLib)
}

// PutOnBottomInChosenOrder asks the player to order the given cards and
// appends them to the bottom of their library in that order: the first ID in
// the chosen ordering ends up just above the previous bottom card, and the
// last ID becomes the new deepest (very bottom) card. The cards must NOT
// currently be in the library — typically the caller has already removed
// them via RemoveTopN.
//
// Internally this reuses Player.ChooseScryPlacement (passing the cards as the
// "top" argument); both the bottom and topOrder slots in the return are
// concatenated into a single bottom-placement order. This lets test players
// script the order via TestGame.ChooseScry(player, ordering, nil).
//
// Powers Oracle phrases of the form "Put the rest on the bottom of your
// library in any order" (Commune with Dinosaurs, Mausoleum Secrets, etc.).
func (g *Game) PutOnBottomInChosenOrder(p Player, cards []Card) {
	if p == nil || len(cards) == 0 {
		return
	}
	snapshot := make([]Card, len(cards))
	copy(snapshot, cards)
	bottom, topOrder := p.ChooseScryPlacement(snapshot, "put on bottom in any order", g)
	bottom, topOrder = validateScryPlacement(snapshot, bottom, topOrder)

	idToCard := make(map[uuid.UUID]Card, len(snapshot))
	for _, c := range snapshot {
		idToCard[c.ID()] = c
	}
	ordered := make([]Card, 0, len(snapshot))
	for _, id := range bottom {
		if c, ok := idToCard[id]; ok {
			ordered = append(ordered, c)
		}
	}
	for _, id := range topOrder {
		if c, ok := idToCard[id]; ok {
			ordered = append(ordered, c)
		}
	}
	lib := p.Library()
	newLib := make([]Card, 0, len(lib)+len(ordered))
	newLib = append(newLib, lib...)
	newLib = append(newLib, ordered...)
	p.SetLibrary(newLib)
}

// PutOnTopInChosenOrder places the given cards on top of the player's
// library in the supplied order (first element becomes the new top card).
// The cards must NOT currently be in the library.
func (g *Game) PutOnTopInChosenOrder(p Player, cards []Card) {
	if p == nil || len(cards) == 0 {
		return
	}
	lib := p.Library()
	newLib := make([]Card, 0, len(lib)+len(cards))
	newLib = append(newLib, cards...)
	newLib = append(newLib, lib...)
	p.SetLibrary(newLib)
}

// RevealHand returns a copy of the owner's hand for inspection by `viewer`.
// The hand is unchanged. The viewer parameter is informational only — the
// engine does not currently track per-player private knowledge — but cards
// often phrase the effect as "target opponent reveals their hand" and we
// preserve that vocabulary for clarity at the call site.
func (g *Game) RevealHand(viewer, owner Player) []Card {
	if owner == nil {
		return nil
	}
	hand := owner.Hand()
	out := make([]Card, len(hand))
	copy(out, hand)
	return out
}

// PickFromHand asks `chooser` to select one card from `owner`'s hand that
// matches `filter`. Returns nil if no card in the hand matches, or if the
// chooser declined (only possible when mayDecline is true). The card is NOT
// removed from the hand; callers move it themselves (e.g. via PlayerDiscard
// or zone-change helpers).
//
// This is the primitive behind "target opponent reveals their hand. You
// choose a [filter] card from it. That player discards that card." (Corpse
// Traders, Entomber Exarch mode 2).
func (g *Game) PickFromHand(chooser Player, owner Player, filter CardFilter, mayDecline bool, reason string) Card {
	if chooser == nil || owner == nil {
		return nil
	}
	hand := owner.Hand()
	var candidates []Card
	for _, c := range hand {
		if filter.Match(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	chosen := chooser.ChooseCardFromLibrary(candidates, reason, g)
	if chosen == nil && !mayDecline {
		chosen = candidates[0]
	}
	return chosen
}
