package main

import (
	"math/rand"

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
		share := (n * count) / total
		if share < 1 {
			share = 1
		}
		if share > remaining {
			share = remaining
		}
		landName := colorToLand(c)
		for i := 0; i < share; i++ {
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
