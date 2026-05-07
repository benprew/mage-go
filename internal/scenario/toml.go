package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseRogueDeck parses a Shandalar rogue deck TOML file.
func ParseRogueDeck(path string) (*RogueDeck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	deck := &RogueDeck{SourceFile: filepath.Base(path)}
	var target *[]DeckEntry
	inArray := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}

		if inArray {
			if line == "]" {
				inArray = false
				target = nil
				continue
			}
			entry, err := parseCardEntry(line)
			if err == nil && target != nil {
				*target = append(*target, entry)
			}
			continue
		}

		if strings.HasPrefix(line, "main_cards") {
			inArray = true
			target = &deck.MainCards
			continue
		}
		if strings.HasPrefix(line, "sideboard_cards") {
			inArray = true
			target = &deck.Sideboard
			continue
		}

		key, val, ok := parseKeyValue(line)
		if !ok {
			continue
		}
		switch key {
		case "name":
			deck.Name = unquote(val)
		case "level":
			deck.Level, _ = strconv.Atoi(val)
		}
	}

	if deck.Name == "" {
		deck.Name = strings.TrimSuffix(filepath.Base(path), ".toml")
	}
	return deck, nil
}

// LoadAllRogueDecks loads all .toml rogue deck files from a directory.
// Unparseable files are skipped with a warning printed to stderr.
func LoadAllRogueDecks(dir string) ([]*RogueDeck, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading rogues dir: %w", err)
	}
	var decks []*RogueDeck
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		deck, err := ParseRogueDeck(filepath.Join(dir, e.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
			continue
		}
		decks = append(decks, deck)
	}
	return decks, nil
}

// parseCardEntry parses a line like: ["4", "Lightning Bolt"],
func parseCardEntry(line string) (DeckEntry, error) {
	line = strings.TrimSuffix(strings.TrimSpace(line), ",")
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		return DeckEntry{}, fmt.Errorf("not a card entry: %q", line)
	}
	line = line[1 : len(line)-1]

	parts := strings.SplitN(line, ",", 2)
	if len(parts) != 2 {
		return DeckEntry{}, fmt.Errorf("expected 2 fields: %q", line)
	}
	countStr := unquote(strings.TrimSpace(parts[0]))
	name := unquote(strings.TrimSpace(parts[1]))

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return DeckEntry{}, fmt.Errorf("bad count %q: %w", countStr, err)
	}
	return DeckEntry{Name: name, Count: count}, nil
}

func parseKeyValue(line string) (key, val string, ok bool) {
	before, after, ok0 := strings.Cut(line, "=")
	if !ok0 {
		return "", "", false
	}
	return strings.TrimSpace(before), strings.TrimSpace(after), true
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
