package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Catalog provides indexed access to card metadata loaded from
// per-set JSON files produced by cmd/fetchcatalog.
type Catalog struct {
	sets  map[string]SetInfo // code → SetInfo
	cards []CardEntry        // flat list of all cards

	// Indexes (keys are lowercased for case-insensitive lookup).
	byName      map[string][]int // lowercase name → card indices
	bySet       map[string][]int // set code → card indices
	bySetNumber map[string]int   // "set:number" → index
	bySetName   map[string]int   // "set:lowername" → index
	byArtist    map[string][]int // lowercase artist → indices
	byOracleID  map[string][]int // oracle_id → indices (all printings)
}

// LoadSet loads a single set's card JSON file and optional set metadata.
// cardFile should be a path like "data/catalog/LEA.json".
// If a matching set metadata file exists at "../_sets/LEA.json" relative
// to cardFile, it is loaded automatically.
func LoadSet(cardFile string) (*Catalog, error) {
	c := newCatalog()
	if err := c.AddSet(cardFile); err != nil {
		return nil, err
	}
	return c, nil
}

// LoadAll loads all set JSON files from dir (e.g. "data/catalog/").
// Set metadata is loaded from dir/_sets/.
func LoadAll(dir string) (*Catalog, error) {
	c := newCatalog()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading catalog dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if err := c.AddSet(filepath.Join(dir, e.Name())); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// AddSet loads a single set's card JSON file into the catalog.
// Set metadata is loaded from the _sets/ subdirectory if present.
func (c *Catalog) AddSet(cardFile string) error {
	data, err := os.ReadFile(cardFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", cardFile, err)
	}

	var cards []CardEntry
	if err := json.Unmarshal(data, &cards); err != nil {
		return fmt.Errorf("parsing %s: %w", cardFile, err)
	}

	// Try to load set metadata.
	base := filepath.Base(cardFile)
	setsDir := filepath.Join(filepath.Dir(cardFile), "_sets")
	setFile := filepath.Join(setsDir, base)
	if setData, err := os.ReadFile(setFile); err == nil {
		var si SetInfo
		if err := json.Unmarshal(setData, &si); err == nil {
			c.sets[strings.ToLower(si.Code)] = si
		}
	}

	// Index cards.
	c.indexCards(cards)

	return nil
}

// addSetFromBytes loads card data from raw JSON bytes into the catalog.
// code is the set code, name is the display name.
func (c *Catalog) addSetFromBytes(code, name string, data []byte) error {
	var cards []CardEntry
	if err := json.Unmarshal(data, &cards); err != nil {
		return fmt.Errorf("parsing %s catalog data: %w", code, err)
	}

	// Store set info from the provided metadata.
	c.sets[strings.ToLower(code)] = SetInfo{Code: code, Name: name}

	// Index cards.
	c.indexCards(cards)
	return nil
}

func (c *Catalog) indexCards(cards []CardEntry) {
	for _, card := range cards {
		idx := len(c.cards)
		c.cards = append(c.cards, card)

		lowName := strings.ToLower(card.Name)
		setCode := strings.ToLower(card.Set)

		c.byName[lowName] = append(c.byName[lowName], idx)
		c.bySet[setCode] = append(c.bySet[setCode], idx)
		c.bySetNumber[setCode+":"+strings.ToLower(card.CollectorNumber)] = idx
		c.bySetName[setCode+":"+lowName] = idx

		if card.Artist != "" {
			lowArtist := strings.ToLower(card.Artist)
			c.byArtist[lowArtist] = append(c.byArtist[lowArtist], idx)
		}

		if card.OracleID != "" {
			c.byOracleID[card.OracleID] = append(c.byOracleID[card.OracleID], idx)
		}

		// Index split card face names (e.g. "Fire // Ice" → "fire", "ice").
		if strings.Contains(card.Name, " // ") {
			parts := strings.SplitSeq(card.Name, " // ")
			for part := range parts {
				lowPart := strings.ToLower(part)
				if lowPart != lowName {
					c.byName[lowPart] = append(c.byName[lowPart], idx)
				}
			}
		}
	}
}

func newCatalog() *Catalog {
	return &Catalog{
		sets:        make(map[string]SetInfo),
		byName:      make(map[string][]int),
		bySet:       make(map[string][]int),
		bySetNumber: make(map[string]int),
		bySetName:   make(map[string]int),
		byArtist:    make(map[string][]int),
		byOracleID:  make(map[string][]int),
	}
}
