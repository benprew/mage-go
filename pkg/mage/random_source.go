package mage

import (
	"math/rand"

	"github.com/google/uuid"
)

// RandomSource encapsulates the generation and deterministic scripting of random
// outcomes (coin flips and uniform integer selections) across games and clones.
type RandomSource struct {
	coinFlipResults []bool
	randomResults   []int
}

// NewRandomSource returns an initialized RandomSource.
func NewRandomSource() RandomSource {
	return RandomSource{}
}

// SetCoinFlipResults sets deterministic coin flip results for testing.
func (r *RandomSource) SetCoinFlipResults(results []bool) {
	r.coinFlipResults = append([]bool(nil), results...)
}

// SetRandomResults sets raw deterministic RandIntn results for testing. Values
// are consumed in order and normalized to each requested range.
func (r *RandomSource) SetRandomResults(results []int) {
	r.randomResults = append([]int(nil), results...)
}

// RandIntn returns a random integer in [0, n). For n <= 0 it returns 0 and
// does not consume a scripted result. SetRandomResults can supply deterministic
// raw values for tests; each value is normalized into range with mathematical
// modulo, so negative scripted values wrap from the end of the range. Once the
// scripted values are exhausted, RandIntn uses the process random source.
func (r *RandomSource) RandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	if len(r.randomResults) == 0 {
		return rand.Intn(n)
	}
	result := r.randomResults[0]
	r.randomResults = r.randomResults[1:]
	result %= n
	if result < 0 {
		result += n
	}
	return result
}

// FlipCoin simulates a coin flip. Returns true for "win" (heads).
// If coinFlipResults is non-empty, pops from the front (for test determinism).
func (r *RandomSource) FlipCoin(_ uuid.UUID) bool {
	if len(r.coinFlipResults) > 0 {
		result := r.coinFlipResults[0]
		r.coinFlipResults = r.coinFlipResults[1:]
		return result
	}
	return r.RandIntn(2) == 0
}

// Clone creates an independent deep copy of RandomSource.
func (r *RandomSource) Clone() RandomSource {
	var coins []bool
	if len(r.coinFlipResults) > 0 {
		coins = append([]bool(nil), r.coinFlipResults...)
	}
	var randoms []int
	if len(r.randomResults) > 0 {
		randoms = append([]int(nil), r.randomResults...)
	}
	return RandomSource{
		coinFlipResults: coins,
		randomResults:   randoms,
	}
}
