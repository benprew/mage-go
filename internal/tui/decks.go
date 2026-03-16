package tui

import (
	"math/rand"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"github.com/google/uuid"
)

// DeckEntry defines a card name and count.
type DeckEntry struct {
	Name  string
	Count int
}

// DeckArchetype is a named deck list for the lobby deck selector.
type DeckArchetype struct {
	Name    string
	Entries []DeckEntry
}

// humanDeckEntries: White/Green aggro with auras and combat tricks.
var humanDeckEntries = []DeckEntry{
	{"Plains", 9},
	{"Forest", 7},
	{"Savannah Lions", 4},
	{"White Knight", 3},
	{"Llanowar Elves", 4},
	{"Grizzly Bears", 3},
	{"Serra Angel", 2},
	{"Healing Salve", 2},
	{"Giant Growth", 2},
	{"Swords to Plowshares", 2},
	{"Holy Strength", 2},
	{"Lure", 1},
	{"Crusade", 1},
}

// aiDeckEntries: Red/Black aggro with burn, removal, auras, and X spell.
var aiDeckEntries = []DeckEntry{
	{"Mountain", 8},
	{"Swamp", 8},
	{"Lightning Bolt", 4},
	{"Terror", 3},
	{"Black Knight", 4},
	{"Ironclaw Orcs", 3},
	{"Hill Giant", 3},
	{"Hypnotic Specter", 3},
	{"Unholy Strength", 1},
	{"Firebreathing", 1},
	{"Disintegrate", 2},
}

// Archetypes is the list of deck archetypes available in the lobby.
var Archetypes = []DeckArchetype{
	{Name: "WG Aggro", Entries: humanDeckEntries},
	{Name: "RB Burn", Entries: aiDeckEntries},
}

// BuildDeck creates a shuffled deck of cards from a deck list, setting card owners.
func BuildDeck(entries []DeckEntry, ownerID uuid.UUID) []mage.Card {
	var deck []mage.Card
	for _, entry := range entries {
		for i := 0; i < entry.Count; i++ {
			card, err := mage.CreateCard(entry.Name)
			if err != nil {
				panic("unknown card: " + entry.Name)
			}
			card.SetOwner(ownerID)
			deck = append(deck, card)
		}
	}
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

// DrawOpeningHand draws 7 cards from the library into the player's hand.
func DrawOpeningHand(p mage.Player) {
	for i := 0; i < 7; i++ {
		p.DrawCard()
	}
}
