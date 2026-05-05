package search

import (
	"testing"
	"unsafe"
)

func TestTTEntry_Size(t *testing.T) {
	// Cache-line budgeting assumes 16-byte entries. If this fires, adjust
	// the entrySize constant in NewTranspositionTable.
	if got := unsafe.Sizeof(TTEntry{}); got != 16 {
		t.Fatalf("TTEntry size = %d, want 16", got)
	}
}

func TestTT_ExactRoundTrip(t *testing.T) {
	tt := NewTranspositionTable(1)
	tt.Store(0xDEADBEEF, 5, 42, TTExact)

	score, ok := tt.Probe(0xDEADBEEF, 5, -1000, 1000)
	if !ok {
		t.Fatalf("expected TT hit on exact match")
	}
	if score != 42 {
		t.Fatalf("expected score 42, got %d", score)
	}
}

func TestTT_MissOnHashMismatch(t *testing.T) {
	tt := NewTranspositionTable(1)
	tt.Store(0xDEADBEEF, 5, 42, TTExact)

	if _, ok := tt.Probe(0xCAFEBABE, 5, -1000, 1000); ok {
		t.Fatalf("expected TT miss for wrong hash")
	}
}

func TestTT_InsufficientDepthMisses(t *testing.T) {
	tt := NewTranspositionTable(1)
	tt.Store(0xABCD, 3, 42, TTExact)

	// Stored depth 3, probing at depth 5 — need deeper info than we have.
	if _, ok := tt.Probe(0xABCD, 5, -1000, 1000); ok {
		t.Fatalf("expected miss when stored depth < probe depth")
	}
	// Stored depth 3, probing at depth 3 — exact match allowed.
	if _, ok := tt.Probe(0xABCD, 3, -1000, 1000); !ok {
		t.Fatalf("expected hit when stored depth == probe depth")
	}
	// Stored depth 3, probing at depth 1 — deeper info suffices.
	if _, ok := tt.Probe(0xABCD, 1, -1000, 1000); !ok {
		t.Fatalf("expected hit when stored depth > probe depth")
	}
}

func TestTT_LowerBoundProbe(t *testing.T) {
	tt := NewTranspositionTable(1)
	tt.Store(0xABCD, 5, 100, TTLowerBound)

	// score >= beta → should hit and cause a beta cutoff.
	if score, ok := tt.Probe(0xABCD, 5, -1000, 50); !ok || score != 100 {
		t.Fatalf("expected lower-bound hit when score >= beta, got ok=%v score=%d", ok, score)
	}
	// score < beta → cannot use the bound.
	if _, ok := tt.Probe(0xABCD, 5, -1000, 200); ok {
		t.Fatalf("expected lower-bound miss when score < beta")
	}
}

func TestTT_UpperBoundProbe(t *testing.T) {
	tt := NewTranspositionTable(1)
	tt.Store(0xABCD, 5, 10, TTUpperBound)

	// score <= alpha → should hit and cause an alpha cutoff.
	if score, ok := tt.Probe(0xABCD, 5, 50, 1000); !ok || score != 10 {
		t.Fatalf("expected upper-bound hit when score <= alpha, got ok=%v score=%d", ok, score)
	}
	// score > alpha → cannot use the bound.
	if _, ok := tt.Probe(0xABCD, 5, -50, 1000); ok {
		t.Fatalf("expected upper-bound miss when score > alpha")
	}
}

func TestTT_AlwaysReplace(t *testing.T) {
	tt := NewTranspositionTable(1)
	// Store at a given hash, then overwrite at the same hash with different data.
	tt.Store(0xABCD, 3, 50, TTExact)
	tt.Store(0xABCD, 7, 99, TTExact)

	score, ok := tt.Probe(0xABCD, 7, -1000, 1000)
	if !ok || score != 99 {
		t.Fatalf("expected overwritten entry, got ok=%v score=%d", ok, score)
	}
}

func TestTT_SizeIsPowerOfTwo(t *testing.T) {
	tt := NewTranspositionTable(4)
	n := tt.Size()
	if n == 0 || n&(n-1) != 0 {
		t.Fatalf("expected power-of-two size, got %d", n)
	}
}

func TestTT_NilSafe(t *testing.T) {
	var tt *TranspositionTable
	// Should not panic.
	tt.Store(1, 1, 1, TTExact)
	if _, ok := tt.Probe(1, 1, 0, 0); ok {
		t.Fatalf("nil TT should never hit")
	}
	tt.Clear()
}

func TestTT_Clear(t *testing.T) {
	tt := NewTranspositionTable(1)
	tt.Store(0xABCD, 5, 42, TTExact)
	tt.Clear()
	if _, ok := tt.Probe(0xABCD, 5, -1000, 1000); ok {
		t.Fatalf("expected miss after Clear")
	}
}
