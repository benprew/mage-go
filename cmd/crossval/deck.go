package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// buildRandomDeck builds a deck of card names from the intersection of cards
// available in both engines. Returns an ordered list of card names.
// Biased toward low-CMC spells for faster, more interesting games.
func buildRandomDeck(available []string, rng *rand.Rand) []string {
	// Separate by CMC bucket: prefer cheap spells
	var cheap, mid, expensive []string // CMC 1-2, 3-4, 5+
	for _, name := range available {
		if isBasicLand(name) {
			continue
		}
		card, err := mage.CreateCard(name)
		if err != nil {
			continue
		}
		cmc := card.ManaCost().CMC()
		if cmc <= 0 {
			continue
		}
		switch {
		case cmc <= 2:
			cheap = append(cheap, name)
		case cmc <= 4:
			mid = append(mid, name)
		default:
			expensive = append(expensive, name)
		}
	}

	// Shuffle each bucket
	rng.Shuffle(len(cheap), func(i, j int) { cheap[i], cheap[j] = cheap[j], cheap[i] })
	rng.Shuffle(len(mid), func(i, j int) { mid[i], mid[j] = mid[j], mid[i] })
	rng.Shuffle(len(expensive), func(i, j int) { expensive[i], expensive[j] = expensive[j], expensive[i] })

	// Build spell list: ~12 cheap, ~8 mid, ~5 expensive
	var deckSpells []string
	counts := make(map[string]int)
	addFrom := func(pool []string, want int) {
		for _, name := range pool {
			if len(deckSpells) >= 25 {
				return
			}
			if want <= 0 {
				return
			}
			if counts[name] >= 4 {
				continue
			}
			deckSpells = append(deckSpells, name)
			counts[name]++
			want--
		}
	}
	addFrom(cheap, 12)
	addFrom(mid, 8)
	addFrom(expensive, 5)

	if len(deckSpells) == 0 {
		deck := make([]string, 40)
		for i := range deck {
			deck[i] = "Plains"
		}
		return deck
	}

	// Figure out what colors we need
	colors := detectColors(deckSpells)

	// Add lands to reach 40 cards
	landCount := 40 - len(deckSpells)
	lands := distributeLands(colors, landCount)

	deck := append(deckSpells, lands...)
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	return deck
}

func isBasicLand(name string) bool {
	switch name {
	case "Plains", "Island", "Swamp", "Mountain", "Forest":
		return true
	}
	return false
}

func detectColors(spells []string) map[core.Color]int {
	colors := make(map[core.Color]int)
	for _, name := range spells {
		card, err := mage.CreateCard(name)
		if err != nil {
			continue
		}
		for _, c := range card.ManaCost().Colors() {
			colors[c]++
		}
	}
	return colors
}

func distributeLands(colors map[core.Color]int, count int) []string {
	if len(colors) == 0 {
		lands := make([]string, count)
		for i := range lands {
			lands[i] = "Plains"
		}
		return lands
	}

	total := 0
	for _, n := range colors {
		total += n
	}

	var lands []string
	remaining := count
	colorOrder := []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}
	for _, c := range colorOrder {
		n, ok := colors[c]
		if !ok {
			continue
		}
		share := min(max((n*count)/total, 1), remaining)
		landName := colorToLand(c)
		for range share {
			lands = append(lands, landName)
		}
		remaining -= share
	}
	// Fill any remaining with the first color's land
	for remaining > 0 {
		for _, c := range colorOrder {
			if _, ok := colors[c]; ok {
				lands = append(lands, colorToLand(c))
				remaining--
				break
			}
		}
	}
	return lands
}

func colorToLand(c core.Color) string {
	switch c {
	case core.White:
		return "Plains"
	case core.Blue:
		return "Island"
	case core.Black:
		return "Swamp"
	case core.Red:
		return "Mountain"
	case core.Green:
		return "Forest"
	}
	return "Plains"
}

// pickDeck returns a rogue deck half the time (if any are loaded), otherwise
// builds a random deck from the available card pool.
func pickDeck(rogues []rogueDeck, available []string, rng *rand.Rand) []string {
	if len(rogues) > 0 && rng.Intn(2) == 0 {
		d := rogues[rng.Intn(len(rogues))]
		deck := make([]string, len(d.cards))
		copy(deck, d.cards)
		rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
		return deck
	}
	return buildRandomDeck(available, rng)
}

type rogueDeck struct {
	name  string
	cards []string
}

// loadRogueDecks loads all .toml rogue deck files from a directory and filters
// each deck to only include cards present in the available set.
func loadRogueDecks(dir string, available []string) ([]rogueDeck, error) {
	avail := make(map[string]bool, len(available))
	for _, c := range available {
		avail[c] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading rogues dir: %w", err)
	}

	var decks []rogueDeck
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name, cards, err := parseRogueTOML(filepath.Join(dir, e.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
			continue
		}

		var filtered []string
		for _, c := range cards {
			if avail[c] {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) < 10 {
			continue
		}

		// Pad with lands if we lost too many spells
		if len(filtered) < 40 {
			colors := detectColors(filtered)
			lands := distributeLands(colors, 40-len(filtered))
			filtered = append(filtered, lands...)
		}

		decks = append(decks, rogueDeck{name: name, cards: filtered})
	}
	return decks, nil
}

// parseRogueTOML parses a Shandalar rogue deck TOML file, returning the deck
// name and an expanded list of card names (one entry per copy).
func parseRogueTOML(path string) (string, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}

	var name string
	var cards []string
	inMainCards := false

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}

		if inMainCards {
			if line == "]" {
				inMainCards = false
				continue
			}
			line = strings.TrimSuffix(line, ",")
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
				continue
			}
			inner := line[1 : len(line)-1]
			parts := strings.SplitN(inner, ",", 2)
			if len(parts) != 2 {
				continue
			}
			countStr := strings.Trim(strings.TrimSpace(parts[0]), "\"")
			cardName := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			count, err := strconv.Atoi(countStr)
			if err != nil {
				continue
			}
			for range count {
				cards = append(cards, cardName)
			}
			continue
		}

		if strings.HasPrefix(line, "main_cards") {
			inMainCards = true
			continue
		}

		if before, after, ok := strings.Cut(line, "="); ok {
			key := strings.TrimSpace(before)
			val := strings.TrimSpace(after)
			if key == "name" {
				name = strings.Trim(val, "\"")
			}
		}
	}

	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), ".toml")
	}
	return name, cards, nil
}
