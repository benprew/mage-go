package main

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// Regression coverage for the buffer/scratch dirty-state bug fixed by
// moving directDirty from encodeScratch to scratchPool.
//
// The bug: under encodeBatchGoPackedParallel a buffer's row R can be
// handled by scratch X in one MageEncodeTokensPacked call and by scratch
// Y in the next, because workers acquire scratches from the per-buffer
// pool. directTokenEmitter.reset uses dirty.cardRefSeen from the
// PREVIOUS run on this row to clear only the slots THAT scratch dirtied.
// When dirty lived on the scratch, scratch X's record reflected only
// X's writes — Y's writes on row R from the intervening call were never
// cleared, so the buffer ended up with stale absolute offsets from Y's
// run that pointed past the new run's row span (or, in the production
// path, got delta-shifted by compactPackedRows into a totally garbage
// position). The fix: dirty is per-buffer (one record shared across
// every scratch that touches that row), so reset always sees the full
// set of slots dirtied since the last call on row R regardless of which
// scratch ran it.
//
// These two tests construct the alternation directly:
//
//   - the "PerScratchDirty" test uses two separate ``directDirtyState``
//     values (one per scratch), which is exactly the pre-fix shape, and
//     asserts that this produces stale leakage in the cardRefPos buffer.
//     If a regression reintroduces per-scratch dirty AND wires the
//     callers to pass the per-scratch record, this test still demonstrates
//     the failure mode.
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
	tables.chooseTargetID = 130001
	tables.choosePlayID = 130002
	tables.useAbilityID = 130003
	tables.chosenID = 130004
	tables.chooseBlockID = 130005
	tables.chooseMayID = 130006
	tables.chooseModeID = 130007
	tables.chooseXDigitID = 130008
	tables.chooseManaSourceID = 130009
	tables.noneID = 130010
	tables.noID = 130011
	tables.yesID = 130012
	tables.numCount = 16
	tables.numIDs = []int32{
		131000, 131001, 131002, 131003,
		131004, 131005, 131006, 131007,
		131008, 131009, 131010, 131011,
		131012, 131013, 131014, 131015,
	}
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
// renderPlanIndex assigns “cardCount“ distinct uuid indices (one per
// battlefield permanent) starting at 0. Callers vary cardCount across
// states so the directTokenEmitter writes a different cardRefSeen
// bitset per call.
func directTestState(cardCount int, name string) (*apiGameState, *apiPending) {
	mkPerm := func(slot int) interactive.PermanentState {
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

	// Single pass-priority option keeps emitOption's source/target gather
	// cheap; we want this test to focus on the place-card-ref path on the
	// battlefield, which is where the bug surfaces.
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
	// One output buffer slot for one batch row but oversized so we can
	// place row writes at arbitrary packedCursor values to exaggerate the
	// stale-residue offset.
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
		packedBlankPos:       make([]int32, int64(slots)*int64(cfg.blankMaxBlanks)),
		packedBlankKind:      make([]int32, int64(slots)*int64(cfg.blankMaxBlanks)),
		packedBlankGroup:     make([]int32, int64(slots)*int64(cfg.blankMaxBlanks)),
		packedBlankGroupKind: make([]int32, int64(slots)*int64(cfg.blankMaxBlanks)),
		packedBlankOptionIdx: make([]int32, int64(slots)*int64(cfg.blankMaxBlanks)),
		packedBlankLegalIDs:  make([]int32, int64(slots)*int64(cfg.blankMaxBlanks)*int64(cfg.blankMaxLegal)),
		packedBlankLegalMask: make([]byte, int64(slots)*int64(cfg.blankMaxBlanks)*int64(cfg.blankMaxLegal)),
		packedBlankOverflow:  make([]int32, slots),
	}
}

// runRotation invokes fillTokenAssemblyDirectPacked three times against
// the SAME row 0 in “view“, alternating two scratches and two card
// rosters. “dirtyForCall“ lets the caller decide whether each call
// gets a per-scratch record (the broken design) or a single shared
// per-buffer record (the fix).
func runRotation(
	t *testing.T,
	view outputViews,
	cfg encodeConfig,
	xScratch, yScratch *encodeScratch,
	dirtyForCall func(call int, scratch *encodeScratch) *directDirtyState,
) (call3SeqLength int32) {
	t.Helper()

	// State1/3: 4 cards. State2: 12 cards. The wider roster makes
	// state2's run dirty card-ref slots 4..11 in addition to 0..3, so
	// state1/3's per-scratch dirty bitsets (which only know about 0..3)
	// can never clean up state2's writes to slots 4..11. With a shared
	// dirty record the cleanup is comprehensive.
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
	// Place call 2 at a packedCursor far from row 0's natural span so
	// any leaked positions land deep past row 0's seq_length and are
	// trivially detectable as OOB.
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

	// Per-scratch dirty: simulate the pre-fix shape with two dirty maps
	// keyed by scratch identity. Each scratch tracks only the slots IT
	// has written to row 0 in past calls.
	xDirty := &directDirtyState{}
	yDirty := &directDirtyState{}
	dirtyForCall := func(_ int, scratch *encodeScratch) *directDirtyState {
		if scratch == xScratch {
			return xDirty
		}
		return yDirty
	}

	rowSpanEnd := runRotation(t, view, cfg, xScratch, yScratch, dirtyForCall)

	// We expect at least one cardRefPos in slots 4..11 (set only by
	// state2 / scratch Y) to retain its absolute-offset position from
	// call 2 — well past row 0's span, demonstrating the bug.
	mcr := int64(cfg.tokenMaxCardRefs)
	stale := 0
	for i := int64(0); i < mcr; i++ {
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
				"either the partial-clear logic was tightened or the test states no longer "+
				"trigger the rotation pattern. cardRefPos=%v",
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

	// Production pattern: a single per-buffer dirty record. Both
	// scratches consult it, so partial-clear on each call sees the full
	// set of slots dirtied by ANY scratch since the last call on row 0.
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
	for i := int64(0); i < mcr; i++ {
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

func TestDirectTokenEncodeTargetedPriorityEmitsTargetBlank(t *testing.T) {
	defer directTestSetUp(t)()
	cfg := directTestCfg()
	cfg.blankMaxBlanks = 4
	cfg.blankMaxLegal = 4
	view := directTestAllocOutputs(cfg)
	scratch := newEncodeScratch()
	dirty := &directDirtyState{}

	sourceID := uuid.New()
	targetID := uuid.New()
	selfPlayer := interactive.PlayerState{
		ID:           uuid.New(),
		Name:         "P0",
		Life:         20,
		Hand:         []interactive.CardState{{ID: sourceID, Name: "DirtyTestSmall"}},
		LibraryCount: 50,
	}
	oppPlayer := interactive.PlayerState{
		ID:   uuid.New(),
		Name: "P1",
		Life: 20,
		Battlefield: []interactive.PermanentState{{
			ID:         targetID,
			Name:       "DirtyTestLarge",
			IsCreature: true,
		}},
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
			Kind:         "cast_spell",
			CardID:       sourceID.String(),
			CardUUID:     sourceID,
			ValidTargets: []apiTarget{{ID: targetID.String(), IDUUID: targetID}},
		}},
	}

	if _, _, err := fillTokenAssemblyDirectPacked(0, 0, state, pending, 0, cfg, view, scratch, dirty); err != nil {
		t.Fatalf("fillTokenAssemblyDirectPacked: %s", err.message)
	}

	tables := getTokenTables()
	if got := view.packedBlankKind[:2]; got[0] != tables.choosePlayID || got[1] != tables.chooseTargetID {
		t.Fatalf("blank kinds = %v, want [%d %d]", got, tables.choosePlayID, tables.chooseTargetID)
	}
	if got := view.packedBlankGroupKind[:2]; got[0] != blankGroupCrossBlank || got[1] != blankGroupPerBlank {
		t.Fatalf("blank group kinds = %v", got)
	}
	if got := view.packedBlankOptionIdx[:2]; got[0] != 0 || got[1] != 0 {
		t.Fatalf("blank option indices = %v, want [0 0]", got)
	}
	targetRef := view.packedCardRefPos[0]
	if targetRef < 0 {
		t.Fatalf("target permanent did not receive card-ref slot 0: %v", view.packedCardRefPos[:4])
	}
	if got, want := view.packedBlankLegalIDs[cfg.blankMaxLegal], tables.cardRefIDs[0]; got != want {
		t.Fatalf("target blank legal id = %d, want %d", got, want)
	}
	if got := view.packedBlankLegalMask[cfg.blankMaxLegal]; got != 1 {
		t.Fatalf("target blank legal mask = %d, want 1", got)
	}
}

func TestDirectTokenEncodeStackTargetPriorityEmitsTargetBlank(t *testing.T) {
	defer directTestSetUp(t)()
	cfg := directTestCfg()
	cfg.blankMaxBlanks = 4
	cfg.blankMaxLegal = 4
	view := directTestAllocOutputs(cfg)
	scratch := newEncodeScratch()
	dirty := &directDirtyState{}

	sourceID := uuid.New()
	stackSourceID := uuid.New()
	selfPlayer := interactive.PlayerState{
		ID:           uuid.New(),
		Name:         "P0",
		Life:         20,
		Hand:         []interactive.CardState{{ID: sourceID, Name: "DirtyTestSmall"}},
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
		Stack: []interactive.StackItemState{{
			ID:   stackSourceID.String(),
			Name: "DirtyTestLarge",
		}},
	}
	pending := &apiPending{
		Kind:      "priority",
		PlayerIdx: 0,
		Options: []apiOption{{
			Kind:         "cast_spell",
			CardID:       sourceID.String(),
			CardUUID:     sourceID,
			ValidTargets: []apiTarget{{ID: stackSourceID.String(), IDUUID: stackSourceID}},
		}},
	}

	if _, _, err := fillTokenAssemblyDirectPacked(0, 0, state, pending, 0, cfg, view, scratch, dirty); err != nil {
		t.Fatalf("fillTokenAssemblyDirectPacked: %s", err.message)
	}

	tables := getTokenTables()
	if got := view.packedBlankKind[:2]; got[0] != tables.choosePlayID || got[1] != tables.chooseTargetID {
		t.Fatalf("blank kinds = %v, want [%d %d]", got, tables.choosePlayID, tables.chooseTargetID)
	}
	if got := view.packedCardRefPos[1]; got < 0 {
		t.Fatalf("stack object did not receive card-ref slot 1: %v", view.packedCardRefPos[:4])
	}
	if got, want := view.packedBlankLegalIDs[cfg.blankMaxLegal], tables.cardRefIDs[1]; got != want {
		t.Fatalf("stack target blank legal id = %d, want %d", got, want)
	}
	if got := view.packedBlankLegalMask[cfg.blankMaxLegal]; got != 1 {
		t.Fatalf("stack target blank legal mask = %d, want 1", got)
	}
}

func TestDirectTokenEncodeChoiceIDsEmitsIndexedChoiceBlank(t *testing.T) {
	defer directTestSetUp(t)()
	cfg := directTestCfg()
	cfg.blankMaxBlanks = 2
	cfg.blankMaxLegal = 4
	view := directTestAllocOutputs(cfg)
	scratch := newEncodeScratch()
	dirty := &directDirtyState{}

	cardA := uuid.New()
	cardB := uuid.New()
	state := &apiGameState{
		Turn:         1,
		Step:         "Precombat Main",
		ActivePlayer: "P0",
		Players: [2]interactive.PlayerState{
			{
				ID:   uuid.New(),
				Name: "P0",
				Life: 20,
				Hand: []interactive.CardState{
					{ID: cardA, Name: "DirtyTestSmall"},
					{ID: cardB, Name: "DirtyTestLarge"},
				},
				LibraryCount: 50,
			},
			{ID: uuid.New(), Name: "P1", Life: 20, LibraryCount: 50},
		},
	}
	pending := &apiPending{
		Kind:      "cards_from_hand",
		PlayerIdx: 0,
		Options: []apiOption{
			{Kind: "choice", ID: cardA.String(), IDUUID: cardA},
			{Kind: "choice", ID: cardB.String(), IDUUID: cardB},
		},
	}

	if _, _, err := fillTokenAssemblyDirectPacked(0, 0, state, pending, 0, cfg, view, scratch, dirty); err != nil {
		t.Fatalf("fillTokenAssemblyDirectPacked: %s", err.message)
	}

	tables := getTokenTables()
	if got := view.packedBlankKind[0]; got != tables.chooseTargetID {
		t.Fatalf("blank kind = %d, want choose-target %d", got, tables.chooseTargetID)
	}
	if got := view.packedBlankGroupKind[0]; got != blankGroupPerBlank {
		t.Fatalf("blank group kind = %d, want PER_BLANK", got)
	}
	if got := view.packedBlankOptionIdx[0]; got != -1 {
		t.Fatalf("blank option index = %d, want -1", got)
	}
	if got, want := view.packedBlankLegalIDs[:2], tables.cardRefIDs[:2]; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("choice legal ids = %v, want %v", got, want)
	}
	if got := view.packedBlankLegalMask[:2]; got[0] != 1 || got[1] != 1 {
		t.Fatalf("choice legal mask prefix = %v, want [1 1]", got)
	}
}

func TestDirectTokenEncodeAttackersEmitPerBlankAttackChoices(t *testing.T) {
	defer directTestSetUp(t)()
	cfg := directTestCfg()
	cfg.blankMaxBlanks = 2
	cfg.blankMaxLegal = 3
	view := directTestAllocOutputs(cfg)
	scratch := newEncodeScratch()
	dirty := &directDirtyState{}

	attackerID := uuid.New()
	state := &apiGameState{
		Turn:         1,
		Step:         "Declare Attackers",
		ActivePlayer: "P0",
		Players: [2]interactive.PlayerState{
			{
				ID:   uuid.New(),
				Name: "P0",
				Life: 20,
				Battlefield: []interactive.PermanentState{{
					ID:         attackerID,
					Name:       "DirtyTestSmall",
					IsCreature: true,
				}},
				LibraryCount: 50,
			},
			{ID: uuid.New(), Name: "P1", Life: 20, LibraryCount: 50},
		},
	}
	pending := &apiPending{
		Kind:      "attackers",
		PlayerIdx: 0,
		Options: []apiOption{{
			Kind:          "attacker",
			PermanentID:   attackerID.String(),
			PermanentUUID: attackerID,
		}},
	}

	if _, _, err := fillTokenAssemblyDirectPacked(0, 0, state, pending, 0, cfg, view, scratch, dirty); err != nil {
		t.Fatalf("fillTokenAssemblyDirectPacked: %s", err.message)
	}

	tables := getTokenTables()
	if got := view.packedBlankKind[0]; got != tables.chooseTargetID {
		t.Fatalf("attacker blank kind = %d, want choose-target %d", got, tables.chooseTargetID)
	}
	if got := view.packedBlankGroupKind[0]; got != blankGroupPerBlank {
		t.Fatalf("attacker blank group kind = %d, want PER_BLANK", got)
	}
	if got := view.packedBlankOptionIdx[0]; got != 0 {
		t.Fatalf("attacker blank option index = %d, want 0", got)
	}
	if got, want := view.packedBlankLegalIDs[:2], []int32{tables.noneID, tables.cardRefIDs[0]}; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("attacker legal ids = %v, want %v", got, want)
	}
}

func TestDirectTokenEncodeBlockersEmitPerBlankBlockChoices(t *testing.T) {
	defer directTestSetUp(t)()
	cfg := directTestCfg()
	cfg.blankMaxBlanks = 2
	cfg.blankMaxLegal = 3
	view := directTestAllocOutputs(cfg)
	scratch := newEncodeScratch()
	dirty := &directDirtyState{}

	blockerID := uuid.New()
	attackerID := uuid.New()
	state := &apiGameState{
		Turn:         1,
		Step:         "Declare Blockers",
		ActivePlayer: "P0",
		Players: [2]interactive.PlayerState{
			{
				ID:   uuid.New(),
				Name: "P0",
				Life: 20,
				Battlefield: []interactive.PermanentState{{
					ID:         attackerID,
					Name:       "DirtyTestLarge",
					IsCreature: true,
					Attacking:  true,
				}},
				LibraryCount: 50,
			},
			{
				ID:   uuid.New(),
				Name: "P1",
				Life: 20,
				Battlefield: []interactive.PermanentState{{
					ID:         blockerID,
					Name:       "DirtyTestSmall",
					IsCreature: true,
				}},
				LibraryCount: 50,
			},
		},
	}
	pending := &apiPending{
		Kind:      "blockers",
		PlayerIdx: 1,
		Options: []apiOption{{
			Kind:          "blocker",
			PermanentID:   blockerID.String(),
			PermanentUUID: blockerID,
			ValidTargets:  []apiTarget{{ID: attackerID.String(), IDUUID: attackerID}},
		}},
	}

	if _, _, err := fillTokenAssemblyDirectPacked(0, 0, state, pending, 1, cfg, view, scratch, dirty); err != nil {
		t.Fatalf("fillTokenAssemblyDirectPacked: %s", err.message)
	}

	tables := getTokenTables()
	if got := view.packedBlankKind[0]; got != tables.chooseBlockID {
		t.Fatalf("blocker blank kind = %d, want choose-block %d", got, tables.chooseBlockID)
	}
	if got := view.packedBlankGroupKind[0]; got != blankGroupPerBlank {
		t.Fatalf("blocker blank group kind = %d, want PER_BLANK", got)
	}
	if got := view.packedBlankOptionIdx[0]; got != 0 {
		t.Fatalf("blocker blank option index = %d, want 0", got)
	}
	if got, want := view.packedBlankLegalIDs[:2], []int32{tables.noneID, tables.cardRefIDs[1]}; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("blocker legal ids = %v, want %v", got, want)
	}
}
