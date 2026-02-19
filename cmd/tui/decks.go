package main

import (
	"math/rand"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
)

// deckList defines card names and counts.
type deckEntry struct {
	name  string
	count int
}

var humanDeck = []deckEntry{
	{"Forest", 8},
	{"Plains", 8},
	{"Grizzly Bears", 4},
	{"Elvish Mystic", 4},
	{"Serra Angel", 4},
	{"Giant Growth", 4},
	{"White Knight", 4},
	{"Savannah Lions", 4},
}

var aiDeck = []deckEntry{
	{"Mountain", 8},
	{"Swamp", 8},
	{"Lightning Bolt", 4},
	{"Black Knight", 4},
	{"Hypnotic Specter", 4},
	{"Goblin Piker", 4},
	{"Ironclaw Orcs", 4},
	{"Doom Blade", 4},
}

// buildDeck creates a shuffled deck of cards from a deck list, setting card owners.
func buildDeck(entries []deckEntry, ownerID uuid.UUID) []mage.Card {
	var deck []mage.Card
	for _, entry := range entries {
		for i := 0; i < entry.count; i++ {
			card, err := mage.CreateCard(entry.name)
			if err != nil {
				panic("unknown card: " + entry.name)
			}
			card.SetOwner(ownerID)
			deck = append(deck, card)
		}
	}
	// Shuffle
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

// drawOpeningHand draws 7 cards from the library into the player's hand.
func drawOpeningHand(p mage.Player) {
	for i := 0; i < 7; i++ {
		p.DrawCard()
	}
}
