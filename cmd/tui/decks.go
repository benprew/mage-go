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

// humanDeck: White/Green aggro with Healing Salve as a key modal spell.
// Against the AI's red burn + black removal, both modes are live:
//   mode 0 (gain 3 life) — recover from Lightning Bolt to the face
//   mode 1 (prevent 3 damage) — save a creature from Terror / Bolt
var humanDeck = []deckEntry{
	{"Plains", 9},
	{"Forest", 7},
	{"Savannah Lions", 4},  // 1/1 for W — fast start
	{"White Knight", 3},    // 2/2 first strike, protection from black
	{"Llanowar Elves", 4},  // mana acceleration
	{"Grizzly Bears", 3},   // reliable 2/2
	{"Serra Angel", 2},     // top-end threat
	{"Healing Salve", 4},   // the modal spell under test
	{"Giant Growth", 2},    // instant pump
	{"Swords to Plowshares", 2}, // premium removal
}

// aiDeck: Red/Black aggro with burn and removal — makes both Healing Salve modes relevant.
var aiDeck = []deckEntry{
	{"Mountain", 8},
	{"Swamp", 8},
	{"Lightning Bolt", 4},   // 3 damage — exactly what Healing Salve prevents
	{"Terror", 4},           // creature removal
	{"Black Knight", 4},     // 2/2 first strike, protection from white
	{"Ironclaw Orcs", 4},    // 2/2 for 1R
	{"Hill Giant", 4},       // 3/3 for 3R — solid threat
	{"Hypnotic Specter", 4}, // flying, discard
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
