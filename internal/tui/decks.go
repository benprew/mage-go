package tui

import (
	"math/rand"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
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

// agaGalneerDeckEntries: BGW midrange (Shandalar "Aga Galneer").
// Gem Bazaar and Ashes to Ashes are not yet implemented; replaced with basics.
var agaGalneerDeckEntries = []DeckEntry{
	{"Swamp", 9},
	{"Forest", 6},
	{"Plains", 10},
	{"Sol Ring", 1},
	{"Unholy Strength", 2},
	{"Erg Raiders", 2},
	{"El-Hajjâj", 2},
	{"Healing Salve", 2},
	{"Stream of Life", 2},
	{"Onulet", 3},
	{"Greed", 2},
	{"Jade Monolith", 2},
	{"Spirit Link", 3},
	{"Serra Angel", 2},
	{"Sengir Vampire", 2},
	{"Giant Growth", 2},
	{"Giant Spider", 2},
	{"Fungusaur", 2},
	{"Savannah Lions", 2},
	{"Hurricane", 2},
}

// guardianOfTheTuskDeckEntries: GW fatties (Shandalar "Guardian of the Tusk").
var guardianOfTheTuskDeckEntries = []DeckEntry{
	{"Forest", 12},
	{"Plains", 12},
	{"Birds of Paradise", 4},
	{"Giant Growth", 4},
	{"Holy Strength", 4},
	{"Desert Twister", 2},
	{"War Mammoth", 4},
	{"Savannah Lions", 3},
	{"Elder Land Wurm", 2},
	{"Tundra Wolves", 3},
	{"Timber Wolves", 3},
	{"Force of Nature", 2},
	{"Divine Transformation", 2},
	{"Colossus of Sardia", 2},
	{"Berserk", 1},
}

// priestessDeckEntries: mono-white walls/defense (Shandalar "Priestess").
// Pikemen is not yet implemented; replaced with +2 Plains.
var priestessDeckEntries = []DeckEntry{
	{"Plains", 25},
	{"Animate Wall", 4},
	{"Reverse Damage", 2},
	{"Healing Salve", 3},
	{"Ivory Cup", 3},
	{"Wall of Swords", 4},
	{"Wall of Spears", 4},
	{"Spirit Link", 3},
	{"Fortified Area", 2},
	{"Benalish Hero", 2},
	{"Blessing", 2},
	{"Mesa Pegasus", 2},
	{"Castle", 2},
	{"Elder Land Wurm", 2},
}

// Archetypes is the list of deck archetypes available in the lobby.
var Archetypes = []DeckArchetype{
	{Name: "WG Aggro", Entries: humanDeckEntries},
	{Name: "RB Burn", Entries: aiDeckEntries},
	{Name: "Aga Galneer (BGW)", Entries: agaGalneerDeckEntries},
	{Name: "Guardian of the Tusk (GW)", Entries: guardianOfTheTuskDeckEntries},
	{Name: "Priestess (W walls)", Entries: priestessDeckEntries},
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
