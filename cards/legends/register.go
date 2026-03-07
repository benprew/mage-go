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

// basicLandNames are the five basic land card names.
var basicLandNames = map[string]bool{
	"Plains": true, "Island": true, "Swamp": true, "Mountain": true, "Forest": true,
}

// isBasicLand returns true if a card is a basic land (by name).
func isBasicLand(c Card) bool {
	return basicLandNames[c.Name()]
}
