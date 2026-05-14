package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var dckCardLineRE = regexp.MustCompile(`^(SB:\s*)?(\d+)\s+\[[^\]]+\]\s+(.+)$`)

// ParseDCKDeck parses a Shandalar .dck deck file.
func ParseDCKDeck(path string) (*RogueDeck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	deck := &RogueDeck{SourceFile: filepath.Base(path)}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if after, ok := strings.CutPrefix(line, "NAME:"); ok {
			deck.Name = strings.TrimSpace(after)
			continue
		}

		matches := dckCardLineRE.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		count, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("bad count in %s: %w", filepath.Base(path), err)
		}
		entry := DeckEntry{Name: strings.TrimSpace(matches[3]), Count: count}
		if matches[1] != "" {
			deck.Sideboard = append(deck.Sideboard, entry)
		} else {
			deck.MainCards = append(deck.MainCards, entry)
		}
	}

	if deck.Name == "" {
		deck.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return deck, nil
}

// LoadAllDCKDecks loads all .dck rogue deck files from a directory.
func LoadAllDCKDecks(dir string) ([]*RogueDeck, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading rogue dck dir: %w", err)
	}
	var decks []*RogueDeck
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".dck") {
			continue
		}
		deck, err := ParseDCKDeck(filepath.Join(dir, e.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
			continue
		}
		decks = append(decks, deck)
	}
	return decks, nil
}
