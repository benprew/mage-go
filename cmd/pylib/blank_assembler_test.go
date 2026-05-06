package main

import (
	"testing"
)

// makeMinimalBlankTokenTables fills the token-table fields needed by the
// inline-blank lookup path. The arity-checked encoder paths aren't exercised
// by these tests, so we leave the bigger spans nil and only populate scalars.
func makeMinimalBlankTokenTables() *tokenTables {
	num := []int32{500, 501, 502, 503, 504, 505, 506, 507, 508, 509, 510, 511, 512, 513, 514, 515}
	return &tokenTables{
		// Inline-blank singletons.
		chooseTargetID:      400,
		chooseBlockID:       401,
		chooseDamageOrderID: 402,
		chooseModeID:        403,
		chooseMayID:         404,
		chooseXDigitID:      405,
		chooseManaSourceID:  406,
		choosePlayID:        407,
		useAbilityID:        408,
		chosenID:            409,
		yesID:               410,
		noID:                411,
		noneID:              412,
		xEndID:              413,
		mulliganID:          414,
		keepID:              415,
		numCount:            int32(len(num)),
		numIDs:              num,
	}
}

func TestBlankSingletonAtRoundTrip(t *testing.T) {
	tab := makeMinimalBlankTokenTables()
	want := []int32{400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415}
	for idx, w := range want {
		got, ok := tab.blankSingletonAt(int32(idx))
		if !ok {
			t.Fatalf("blankSingletonAt(%d): not ok", idx)
		}
		if got != w {
			t.Fatalf("blankSingletonAt(%d): got %d want %d", idx, got, w)
		}
	}
	if _, ok := tab.blankSingletonAt(16); ok {
		t.Fatalf("blankSingletonAt(16): expected not ok")
	}
	if _, ok := tab.blankSingletonAt(-1); ok {
		t.Fatalf("blankSingletonAt(-1): expected not ok")
	}
}

func TestBlankNumIDsAccessor(t *testing.T) {
	tab := makeMinimalBlankTokenTables()
	if tab.numCount != 16 {
		t.Fatalf("numCount: got %d want 16", tab.numCount)
	}
	for k := int32(0); k < tab.numCount; k++ {
		if got := tab.numIDs[k]; got != 500+k {
			t.Fatalf("numIDs[%d]: got %d want %d", k, got, 500+k)
		}
	}
}

func newBlankCollector(maxBlanks, vmax int32) *blankCollector {
	c := &blankCollector{
		positions: make([]int32, maxBlanks),
		kind:      make([]int32, maxBlanks),
		group:     make([]int32, maxBlanks),
		groupKind: make([]int32, maxBlanks),
		optionIdx: make([]int32, maxBlanks),
		legalIDs:  make([]int32, maxBlanks*vmax),
		legalMask: make([]uint8, maxBlanks*vmax),
	}
	c.reset(maxBlanks, vmax)
	return c
}

// TestEmitBlankWalker drives a small render plan with three EMIT_BLANK
// opcodes (one per group-kind, varying legal-list sizes) and asserts the
// blank-collector outputs are populated correctly.
func TestEmitBlankWalker(t *testing.T) {
	plan := []int32{
		// Some unrelated structural opcode — the blank walker must skip it.
		opOpenActions,
		// Blank 0: choose_target, group 0, PER_BLANK, 3 legal ids.
		opEmitBlank, 400, 0, blankGroupPerBlank, 3,
		opEmitBlankLegal, 1000,
		opEmitBlankLegal, 1001,
		opEmitBlankLegal, 1002,
		// Blank 1: choose_block, group 7, CROSS_BLANK, 2 legal ids.
		opEmitBlank, 401, 7, blankGroupCrossBlank, 2,
		opEmitBlankLegal, 2000,
		opEmitBlankLegal, 2001,
		// Another structural opcode in between.
		opCloseActions,
		// Blank 2: choose_x_digit, group 9, CONSTRAINED, 1 legal id.
		opEmitBlank, 405, 9, blankGroupConstrained, 1,
		opEmitBlankLegal, 3000,
	}

	const K = 4
	const Vmax = 4
	col := newBlankCollector(K, Vmax)

	// Use a counter that simulates a token cursor advancing as the walker
	// passes each blank. In production the cursor comes from the token
	// emitter; for the test we increment by 100 per blank to make the
	// recorded positions easy to verify.
	cursor := int32(0)
	cursorFn := func() int32 {
		cursor += 100
		return cursor
	}

	if err := walkBlankPlan(plan, col, cursorFn); err != nil {
		t.Fatalf("walkBlankPlan: %v", err)
	}

	if col.blankCount != 3 {
		t.Fatalf("blankCount: got %d want 3", col.blankCount)
	}
	wantPos := []int32{100, 200, 300}
	wantKind := []int32{400, 401, 405}
	wantGroup := []int32{0, 7, 9}
	wantGroupKind := []int32{blankGroupPerBlank, blankGroupCrossBlank, blankGroupConstrained}
	for i := 0; i < 3; i++ {
		if col.positions[i] != wantPos[i] {
			t.Errorf("positions[%d] = %d, want %d", i, col.positions[i], wantPos[i])
		}
		if col.kind[i] != wantKind[i] {
			t.Errorf("kind[%d] = %d, want %d", i, col.kind[i], wantKind[i])
		}
		if col.group[i] != wantGroup[i] {
			t.Errorf("group[%d] = %d, want %d", i, col.group[i], wantGroup[i])
		}
		if col.groupKind[i] != wantGroupKind[i] {
			t.Errorf("groupKind[%d] = %d, want %d", i, col.groupKind[i], wantGroupKind[i])
		}
		if col.optionIdx[i] != -1 {
			t.Errorf("optionIdx[%d] = %d, want -1", i, col.optionIdx[i])
		}
	}
	// Slot K=3 was never written; should still be sentinel (-1 for positions).
	if col.positions[3] != -1 {
		t.Errorf("positions[3] = %d, want -1 (unused)", col.positions[3])
	}

	// Legal ids row-major: row k, col v -> legalIDs[k*Vmax + v].
	type legalCheck struct {
		k, v int32
		id   int32
		mask uint8
	}
	checks := []legalCheck{
		{0, 0, 1000, 1}, {0, 1, 1001, 1}, {0, 2, 1002, 1}, {0, 3, 0, 0},
		{1, 0, 2000, 1}, {1, 1, 2001, 1}, {1, 2, 0, 0}, {1, 3, 0, 0},
		{2, 0, 3000, 1}, {2, 1, 0, 0}, {2, 2, 0, 0}, {2, 3, 0, 0},
		{3, 0, 0, 0},
	}
	for _, c := range checks {
		idx := c.k*Vmax + c.v
		if col.legalIDs[idx] != c.id {
			t.Errorf("legalIDs[%d,%d] = %d, want %d", c.k, c.v, col.legalIDs[idx], c.id)
		}
		if col.legalMask[idx] != c.mask {
			t.Errorf("legalMask[%d,%d] = %d, want %d", c.k, c.v, col.legalMask[idx], c.mask)
		}
	}
}

func TestEmitBlankLegalCountMismatch(t *testing.T) {
	// Declares 2 legal ids but only one EMIT_BLANK_LEGAL follows.
	plan := []int32{
		opEmitBlank, 400, 0, blankGroupPerBlank, 2,
		opEmitBlankLegal, 1000,
	}
	col := newBlankCollector(4, 4)
	err := walkBlankPlan(plan, col, nil)
	if err == nil {
		t.Fatalf("expected error for legal_count mismatch, got nil")
	}
}

func TestEmitBlankLegalOverflow(t *testing.T) {
	// Declares 1 legal id but two EMIT_BLANK_LEGAL ops follow.
	plan := []int32{
		opEmitBlank, 400, 0, blankGroupPerBlank, 1,
		opEmitBlankLegal, 1000,
		opEmitBlankLegal, 1001,
	}
	col := newBlankCollector(4, 4)
	err := walkBlankPlan(plan, col, nil)
	if err == nil {
		t.Fatalf("expected error for legal-id overflow, got nil")
	}
}

// TestAssembleTokensSkipsBlankOpcodes ensures the token-emission walker
// ignores EMIT_BLANK / EMIT_BLANK_LEGAL opcodes. The plan emits them between
// other structural opcodes; the assembler must walk past them without
// erroring on "unknown opcode".
func TestAssembleTokensSkipsBlankOpcodes(t *testing.T) {
	tab := benchTokenTables()
	plan := []int32{
		opOpenState,
		opEmitBlank, tab.chooseTargetID, 0, blankGroupPerBlank, 1,
		opEmitBlankLegal, 9999,
		opCloseState,
	}
	out := tokenAssemblerOut{
		tokenIDs:    make([]int32, 32),
		optionPos:   make([]int32, 1),
		optionMask:  make([]uint8, 1),
		targetPos:   make([]int32, 1),
		targetMask:  make([]uint8, 1),
		cardRefPos:  make([]int32, 1),
		maxOptions:  1,
		maxTargets:  1,
		maxCardRefs: 1,
	}
	if _, _, err := assembleTokensFromPlan(plan, tab, &out, 32); err != nil {
		t.Fatalf("assembleTokensFromPlan: unexpected error %v", err)
	}
}
