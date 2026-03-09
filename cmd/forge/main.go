// cmd/forge generates complete, balanced Magic: The Gathering card sets.
//
// Ported from mage-swift's MageForge. Fully self-contained — no dependency
// on pkg/mage or any external packages. Seeded RNG for reproducible output.
//
// Usage:
//
//	go run ./cmd/forge [seed] [card-count] [set-name]
//
// Examples:
//
//	go run ./cmd/forge                    # random seed, 200 cards
//	go run ./cmd/forge 42                 # deterministic, 200 cards
//	go run ./cmd/forge 42 200 "Forged Realms"
package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	var seed int64
	count := 200
	name := "Forged Realms"

	args := os.Args[1:]

	if len(args) >= 1 {
		if s, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			seed = s
		} else {
			seed = time.Now().UnixNano()
		}
	} else {
		seed = time.Now().UnixNano()
	}

	if len(args) >= 2 {
		if c, err := strconv.Atoi(args[1]); err == nil && c > 0 {
			count = c
		}
	}

	if len(args) >= 3 {
		name = args[2]
	}

	code := name
	if len(code) > 3 {
		code = code[:3]
	}
	for len(code) < 3 {
		code += "X"
	}

	config := SetConfig{
		CardCount:          count,
		IncludeArtifacts:   true,
		ArtifactPercentage: 0.08,
		SetName:            name,
		SetCode:            upper(code),
	}

	set := GenerateSet(config, seed)

	PrintSummary(set)
	fmt.Printf("  Seed: %d\n", seed)
	fmt.Printf("  Rerun: go run ./cmd/forge %d\n", seed)
	fmt.Println()
	PrintCardList(set)
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

// SeededRNG wraps a math/rand.Rand for reproducible generation.
// Unlike the Swift version's xorshift, we use Go's standard PCG via rand.New.
func NewRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}
