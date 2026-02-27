package legends

import . "github.com/mage/mage/pkg/mage"

const expansionName = "Legends"

// withExpansion wraps a card factory to set the expansion name on all cards.
func withExpansion(factory func() Card) func() Card {
	return func() Card {
		card := factory()
		if bc, ok := card.(*BaseCard); ok {
			WithExpansion(expansionName)(bc)
		}
		return card
	}
}
