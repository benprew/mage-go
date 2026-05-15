package main

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// Regression coverage for the buffer/scratch dirty-state bug fixed by
// moving directDirty from encodeScratch to scratchPool.
//
// Two tests construct an alternating-scratch rotation against the same
// row 0:
//
//   - the "PerScratchDirty" test uses two separate ``directDirtyState``
//     values (one per scratch), which is the pre-fix shape, and asserts
//     that this produces stale leakage in the cardRefPos buffer.
//
//   - the "SharedDirty" test uses the production pattern (one dirty from
//     scratchPool.rowDirty(0)) and asserts the buffer is consistent.

const directTestMaxTokens int32 = 4096

// directTestSetUp registers the bench token tables and card-row overrides
// for the lifetime of the test, returning a cleanup func that restores
// the previous global state so the rest of the test suite isn't perturbed.
func directTestSetUp(t *testing.T) func() {
	t.Helper()
	tables := benchTokenTables()
	tokenTablesMu.Lock()
	prevTables := currentTokenTables
	currentTokenTables = tables
	tokenTablesMu.Unlock()

	cardRowOverrideMu.Lock()
	prevOverrides := cardRowOverrides
	prevOverridesRaw := cardRowOverridesByRaw
	prevOverridden := cardRowsOverridden
	cardRowOverrideMu.Unlock()

	rows := make(map[string]int64)
	rows["DirtyTestSmall"] = 1
	rows["DirtyTestLarge"] = 2
	setCardRowOverrides(rows)

	return func() {
		tokenTablesMu.Lock()
		currentTokenTables = prevTables
		tokenTablesMu.Unlock()
		cardRowOverrideMu.Lock()
		cardRowOverrides = prevOverrides
		cardRowOverridesByRaw = prevOverridesRaw
		cardRowsOverridden = prevOverridden
		cardRowOverrideMu.Unlock()
	}
}

// directTestState builds a simple priority-pending snapshot whose
// renderPlanIndex assigns ``cardCount`` distinct uuid indices (one per
// battlefield permanent) starting at 0.
func directTestState(cardCount int, name string) (*apiGameState, *apiPending) {
	mkPerm := func(_ int) interactive.PermanentState {
		return interactive.PermanentState{
			ID:         uuid.New(),
			Name:       name,
			Power:      1,
			Toughness:  1,
			IsCreature: true,
		}
	}
	bf := make([]interactive.PermanentState, cardCount)
	for i := range bf {
		bf[i] = mkPerm(i)
	}
	selfPlayer := interactive.PlayerState{
		ID:           uuid.New(),
		Name:         "P0",
		Life:         20,
		Battlefield:  bf,
		LibraryCount: 50,
	}
	oppPlayer := interactive.PlayerState{
		ID:           uuid.New(),
		Name:         "P1",
		Life:         20,
		LibraryCount: 50,
	}
	state := &apiGameState{
		Turn:         1,
		Step:         "Precombat Main",
		ActivePlayer: "P0",
		Players:      [2]interactive.PlayerState{selfPlayer, oppPlayer},
	}
	pending := &apiPending{
		Kind:      "priority",
		PlayerIdx: 0,
		Options: []apiOption{{
			Kind: "pass",
		}},
	}
	return state, pending
}

func directTestCfg() encodeConfig {
	return encodeConfig{
		maxOptions:          8,
		maxTargetsPerOption: 4,
		tokenMaxTokens:      directTestMaxTokens,
		tokenMaxOptions:     8,
		tokenMaxTargets:     4,
		tokenMaxCardRefs:    64,
		dedupCardBodies:     true,
	}
}

func directTestAllocOutputs(cfg encodeConfig) outputViews {
	const slots = 4
	mt := cfg.tokenMaxTokens
	mcr := cfg.tokenMaxCardRefs
	return outputViews{
		packedTokenIDs:       make([]int32, int64(slots)*int64(mt)),
		packedCuSeqlens:      make([]int32, slots+1),
		packedSeqLengths:     make([]int32, slots),
		packedStatePositions: make([]int32, slots),
		packedCardRefPos:     make([]int32, int64(slots)*int64(mcr)),
		packedTokenOverflow:  make([]int32, slots),
	}
}

// runRotation invokes fillTokenAssemblyDirectPacked three times against
// the SAME row 0 in ``view``, alternating two scratches and two card
// rosters. ``dirtyForCall`` lets the caller decide whether each call gets
// a per-scratch record (the broken design) or a single shared per-buffer
// record (the fix).
func runRotation(
	t *testing.T,
	view outputViews,
	cfg encodeConfig,
	xScratch, yScratch *encodeScratch,
	dirtyForCall func(call int, scratch *encodeScratch) *directDirtyState,
) (call3SeqLength int32) {
	t.Helper()

	state1, pending1 := directTestState(4, "DirtyTestSmall")
	state2, pending2 := directTestState(12, "DirtyTestLarge")
	state3, pending3 := directTestState(4, "DirtyTestSmall")

	xScratch.reset()
	if _, _, err := fillTokenAssemblyDirectPacked(
		0, 0, state1, pending1, 0, cfg, view, xScratch, dirtyForCall(1, xScratch),
	); err != nil {
		t.Fatalf("call 1: %s", err.message)
	}
	yScratch.reset()
	if _, _, err := fillTokenAssemblyDirectPacked(
		0, 2*cfg.tokenMaxTokens, state2, pending2, 0, cfg, view, yScratch, dirtyForCall(2, yScratch),
	); err != nil {
		t.Fatalf("call 2: %s", err.message)
	}
	xScratch.reset()
	if _, _, err := fillTokenAssemblyDirectPacked(
		0, 0, state3, pending3, 0, cfg, view, xScratch, dirtyForCall(3, xScratch),
	); err != nil {
		t.Fatalf("call 3: %s", err.message)
	}

	return view.packedSeqLengths[0]
}

func TestDirectTokenEncodePerScratchDirtyLeavesResidue(t *testing.T) {
	defer directTestSetUp(t)()
	cfg := directTestCfg()
	view := directTestAllocOutputs(cfg)

	xScratch := newEncodeScratch()
	yScratch := newEncodeScratch()

	xDirty := &directDirtyState{}
	yDirty := &directDirtyState{}
	dirtyForCall := func(_ int, scratch *encodeScratch) *directDirtyState {
		if scratch == xScratch {
			return xDirty
		}
		return yDirty
	}

	rowSpanEnd := runRotation(t, view, cfg, xScratch, yScratch, dirtyForCall)

	mcr := int64(cfg.tokenMaxCardRefs)
	stale := 0
	for i := range mcr {
		pos := view.packedCardRefPos[i]
		if pos < 0 {
			continue
		}
		if pos >= rowSpanEnd {
			stale++
		}
	}
	if stale == 0 {
		t.Fatalf(
			"per-scratch dirty: expected stale residue past rowSpanEnd=%d, found none. "+
				"cardRefPos=%v",
			rowSpanEnd, view.packedCardRefPos[:mcr],
		)
	}
}

func TestDirectTokenEncodeSharedDirtyAcrossScratches(t *testing.T) {
	defer directTestSetUp(t)()
	cfg := directTestCfg()
	view := directTestAllocOutputs(cfg)

	xScratch := newEncodeScratch()
	yScratch := newEncodeScratch()

	poolKey := scratchPoolKey(view)
	pool := scratchPoolFor(poolKey)
	pool.mu.Lock()
	pool.ensureDirty(1)
	pool.mu.Unlock()
	dirtyForCall := func(_ int, _ *encodeScratch) *directDirtyState {
		return pool.rowDirty(0)
	}

	rowSpanEnd := runRotation(t, view, cfg, xScratch, yScratch, dirtyForCall)

	mcr := int64(cfg.tokenMaxCardRefs)
	for i := range mcr {
		pos := view.packedCardRefPos[i]
		if pos < 0 {
			continue
		}
		if pos < 0 || pos >= rowSpanEnd {
			t.Errorf(
				"shared dirty: cardRefPos[%d]=%d out of row 0 span [0, %d). "+
					"residue from earlier call leaked through partial-clear.",
				i, pos, rowSpanEnd,
			)
		}
	}
}

func TestCardRowForNameAllowsAbilitySentinelWithOverrides(t *testing.T) {
	defer directTestSetUp(t)()

	row, ok := cardRowForName("Ability")
	if !ok {
		t.Fatalf("cardRowForName(Ability) failed under overrides")
	}
	if row != 0 {
		t.Fatalf("cardRowForName(Ability) = %d, want row 0 unknown-card sentinel", row)
	}
}

func TestBuildBlockerPendingDropsConfirmPassOption(t *testing.T) {
	blockerID := uuid.New()
	attackerID := uuid.New()
	pending := buildBlockerPending(&interactive.GameMsg{
		Options: []interactive.ActionOption{
			{
				Type:         interactive.ActionSelectBlockers,
				Label:        "Blocker",
				PermanentID:  blockerID,
				ValidTargets: []uuid.UUID{attackerID},
			},
			{
				Type:  interactive.ActionPass,
				Label: "Done (confirm blocks)",
			},
		},
	}, 1)

	if got := len(pending.Options); got != 1 {
		t.Fatalf("blocker pending option count = %d, want 1", got)
	}
	if got := pending.Options[0].PermanentUUID; got != blockerID {
		t.Fatalf("blocker pending permanent = %s, want %s", got, blockerID)
	}
}
