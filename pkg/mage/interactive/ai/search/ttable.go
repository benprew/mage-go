package search

// TTFlag identifies the bound information carried by a transposition-table
// entry.
//
//   - TTExact: score is the exact minimax value of this position.
//   - TTLowerBound: score is a lower bound (a beta cutoff occurred — the true
//     value is at least this high).
//   - TTUpperBound: score is an upper bound (no move raised alpha — the true
//     value is at most this high).
type TTFlag uint8

const (
	TTExact TTFlag = iota
	TTLowerBound
	TTUpperBound
)

// TTEntry is a single transposition-table slot. Keeping entries small (16
// bytes on 64-bit targets) lets a few hundred thousand entries fit in L3.
type TTEntry struct {
	hash  uint64
	score int16
	depth uint8
	flag  TTFlag
}

// TranspositionTable is a fixed-size direct-mapped cache of search results
// keyed by Zobrist hash. Collisions overwrite the existing entry
// (always-replace policy), which is simple, lock-free-friendly, and robust
// to search iteration resets.
type TranspositionTable struct {
	entries []TTEntry
	mask    uint64
}

// NewTranspositionTable allocates a TT sized to approximately sizeMB
// megabytes, rounded down to the nearest power of two number of entries.
// sizeMB values <= 0 fall back to 1 MB.
func NewTranspositionTable(sizeMB int) *TranspositionTable {
	if sizeMB <= 0 {
		sizeMB = 1
	}
	const entrySize = 16
	target := uint64(sizeMB) * 1024 * 1024 / entrySize
	n := uint64(1)
	for n*2 <= target {
		n *= 2
	}
	return &TranspositionTable{
		entries: make([]TTEntry, n),
		mask:    n - 1,
	}
}

// Store records a search result for the given position hash. Uses an
// always-replace policy: the new entry clobbers whatever was in the slot,
// regardless of its depth.
func (tt *TranspositionTable) Store(hash uint64, depth, score int, flag TTFlag) {
	if tt == nil {
		return
	}
	s := score
	if s > 32767 {
		s = 32767
	} else if s < -32768 {
		s = -32768
	}
	d := depth
	if d < 0 {
		d = 0
	} else if d > 255 {
		d = 255
	}
	tt.entries[hash&tt.mask] = TTEntry{
		hash:  hash,
		score: int16(s),
		depth: uint8(d),
		flag:  flag,
	}
}

// Probe looks up a TT entry for the given hash. It returns a usable score
// only when:
//
//  1. The slot holds an entry for this hash (no collision).
//  2. The stored depth is at least the remaining search depth.
//  3. The stored flag and bounds permit returning the score:
//     - TTExact: always usable.
//     - TTLowerBound: usable if score >= beta (causes a beta cutoff).
//     - TTUpperBound: usable if score <= alpha (causes an alpha cutoff).
//
// The second return value is true iff a usable score was found.
func (tt *TranspositionTable) Probe(hash uint64, depth, alpha, beta int) (int, bool) {
	if tt == nil {
		return 0, false
	}
	e := tt.entries[hash&tt.mask]
	if e.hash != hash {
		return 0, false
	}
	if int(e.depth) < depth {
		return 0, false
	}
	score := int(e.score)
	switch e.flag {
	case TTExact:
		return score, true
	case TTLowerBound:
		if score >= beta {
			return score, true
		}
	case TTUpperBound:
		if score <= alpha {
			return score, true
		}
	}
	return 0, false
}

// Clear wipes all TT entries. Use between unrelated games; within a game,
// always-replace handles staleness automatically.
func (tt *TranspositionTable) Clear() {
	if tt == nil {
		return
	}
	for i := range tt.entries {
		tt.entries[i] = TTEntry{}
	}
}

// Size returns the number of entry slots (always a power of two).
func (tt *TranspositionTable) Size() int {
	if tt == nil {
		return 0
	}
	return len(tt.entries)
}
