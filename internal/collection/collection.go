// Package collection manages per-player card collections, deck persistence,
// and booster pack timing. It has no bubbletea dependency — the builder
// sub-package adds the UI layer.
package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mage/mage/internal/worlddata"
)

// SavedDeck is a named deck with a main deck list and an optional sideboard.
type SavedDeck struct {
	Name      string
	Main      []worlddata.DeckEntry
	Sideboard []worlddata.DeckEntry
}

// Collection is a player's persistent game state.
type Collection struct {
	// Cards maps card name → number of copies owned.
	Cards map[string]int

	// Gold is the player's current gold balance.
	Gold int

	// LastBooster is when the player last claimed a booster pack.
	LastBooster time.Time

	// Decks is the list of saved named decks.
	Decks []SavedDeck

	// ActiveDeck is the index into Decks of the currently selected deck.
	ActiveDeck int
}

// New returns an empty collection with no cards, no gold, and no decks.
func New() *Collection {
	return &Collection{
		Cards: make(map[string]int),
	}
}

// Add adds one copy of each card name in the list to the collection.
func (c *Collection) Add(cards []string) {
	for _, name := range cards {
		c.Cards[name]++
	}
}

// AddDeckEntries adds cards from a deck entry list (used for starter chest).
func (c *Collection) AddDeckEntries(entries []worlddata.DeckEntry) {
	for _, e := range entries {
		c.Cards[e.Name] += e.Count
	}
}

// Remove removes one copy of the named card from the collection.
// Returns false if the card is not owned.
func (c *Collection) Remove(card string) bool {
	if c.Cards[card] <= 0 {
		return false
	}
	c.Cards[card]--
	if c.Cards[card] == 0 {
		delete(c.Cards, card)
	}
	return true
}

// SpendGold deducts amount from Gold. Returns false if insufficient.
func (c *Collection) SpendGold(amount int) bool {
	if c.Gold < amount {
		return false
	}
	c.Gold -= amount
	return true
}

// AddGold adds amount to Gold.
func (c *Collection) AddGold(amount int) {
	c.Gold += amount
}

// CanClaimBooster returns true if 24 hours have elapsed since LastBooster.
func (c *Collection) CanClaimBooster() bool {
	return time.Since(c.LastBooster) >= 24*time.Hour
}

// ClaimBooster records that a booster was claimed now.
func (c *Collection) ClaimBooster() {
	c.LastBooster = time.Now()
}

// ActiveSavedDeck returns a pointer to the active deck, or nil if none.
func (c *Collection) ActiveSavedDeck() *SavedDeck {
	if len(c.Decks) == 0 {
		return nil
	}
	idx := c.ActiveDeck
	if idx < 0 || idx >= len(c.Decks) {
		idx = 0
	}
	return &c.Decks[idx]
}

// HasDeck returns true if the player has at least one saved deck.
func (c *Collection) HasDeck() bool {
	return len(c.Decks) > 0 && c.ActiveSavedDeck() != nil
}

// dataPath returns the path to the player's collection.json.
func dataPath(dataDir, fingerprint string) string {
	return filepath.Join(dataDir, "players", fingerprint, "collection.json")
}

// Load reads a player's collection from disk.
// Returns a new empty Collection if the file does not exist.
func Load(dataDir, fingerprint string) (*Collection, error) {
	path := dataPath(dataDir, fingerprint)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return nil, err
	}
	var col Collection
	if err := json.Unmarshal(data, &col); err != nil {
		return nil, err
	}
	if col.Cards == nil {
		col.Cards = make(map[string]int)
	}
	return &col, nil
}

// Save writes the collection to disk, creating parent directories as needed.
func (c *Collection) Save(dataDir, fingerprint string) error {
	path := dataPath(dataDir, fingerprint)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
