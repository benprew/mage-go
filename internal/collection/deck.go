package collection

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/mage/mage/internal/worlddata"
	"github.com/mage/mage/pkg/mage"
)

const (
	MinDeckSize     = 20
	MaxCopiesPerCard = 4
)

// BuildDeck instantiates a shuffled slice of mage.Card from a deck entry list.
// Cards are owned by ownerID. Returns an error if any card name is unknown.
func BuildDeck(entries []worlddata.DeckEntry, ownerID uuid.UUID) ([]mage.Card, error) {
	var deck []mage.Card
	for _, e := range entries {
		for i := 0; i < e.Count; i++ {
			card, err := mage.CreateCard(e.Name)
			if err != nil {
				return nil, fmt.Errorf("unknown card %q: %w", e.Name, err)
			}
			card.SetOwner(ownerID)
			deck = append(deck, card)
		}
	}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck, nil
}

// Validate checks that deck is legal given the player's collection.
// Returns a slice of human-readable error strings; empty means valid.
func Validate(deck SavedDeck, coll *Collection) []string {
	var errs []string

	total := 0
	for _, e := range deck.Main {
		total += e.Count
	}
	if total < MinDeckSize {
		errs = append(errs, fmt.Sprintf("deck has %d cards (minimum %d)", total, MinDeckSize))
	}

	// Count copies across main + sideboard combined, check against collection.
	copies := make(map[string]int)
	for _, e := range deck.Main {
		copies[e.Name] += e.Count
	}
	for _, e := range deck.Sideboard {
		copies[e.Name] += e.Count
	}

	for name, count := range copies {
		if !isBasicLand(name) && count > MaxCopiesPerCard {
			errs = append(errs, fmt.Sprintf("too many copies of %q (%d, max %d)", name, count, MaxCopiesPerCard))
		}
		if coll.Cards[name] < count {
			errs = append(errs, fmt.Sprintf("not enough %q (have %d, need %d)", name, coll.Cards[name], count))
		}
	}
	return errs
}

// DrawAnteCard removes a random card from deck and returns it with the
// shortened deck. Panics if deck is empty.
func DrawAnteCard(deck []mage.Card) (mage.Card, []mage.Card) {
	i := rand.Intn(len(deck))
	card := deck[i]
	deck = append(deck[:i], deck[i+1:]...)
	return card, deck
}

func isBasicLand(name string) bool {
	switch name {
	case "Plains", "Island", "Swamp", "Mountain", "Forest":
		return true
	}
	return false
}
