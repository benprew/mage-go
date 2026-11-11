package ai

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// MulliganLondon redraws seven cards at most twice, then puts one card per
// mulligan on the bottom of the library. The count includes any prior mulligans.
func MulliganLondon(player *mage.BasePlayer, mulligans int) {
	mulligans = min(max(mulligans, 0), 7)
	for mulligans < 2 && len(player.Hand()) == 7 {
		lands := 0
		for _, card := range player.Hand() {
			if card.HasType(core.TypeLand) {
				lands++
			}
		}
		if lands >= 2 && lands <= 5 {
			break
		}
		for _, card := range player.Hand() {
			if moved, ok := player.RemoveFromHand(card.ID()); ok {
				player.AddToLibrary(moved)
			}
		}
		player.ShuffleLibrary()
		for range 7 {
			player.DrawCard()
		}
		mulligans++
	}
	for _, card := range LondonBottomCards(player.Hand(), mulligans) {
		if moved, ok := player.RemoveFromHand(card.ID()); ok {
			player.AddToLibrary(moved)
		}
	}
}

// LondonBottomCards selects excess lands and expensive spells to put on the
// bottom. It keeps two lands when enough other cards are available.
func LondonBottomCards(hand []*mage.Card, amount int) []*mage.Card {
	remaining := slices.Clone(hand)
	count := min(max(amount, 0), len(hand))
	bottom := make([]*mage.Card, 0, count)
	for range count {
		lands := 0
		for _, card := range remaining {
			if card.HasType(core.TypeLand) {
				lands++
			}
		}
		index := -1
		for i, card := range remaining {
			if card.HasType(core.TypeLand) {
				if lands > 3 {
					index = i
					break
				}
				continue
			}
			if index < 0 || card.ManaCost().CMC() > remaining[index].ManaCost().CMC() {
				index = i
			}
		}
		if index < 0 {
			index = 0
		}
		bottom = append(bottom, remaining[index])
		remaining = slices.Delete(remaining, index, index+1)
	}
	return bottom
}
