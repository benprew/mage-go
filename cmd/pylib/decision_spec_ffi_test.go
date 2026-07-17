package main

import (
	"testing"

	"github.com/google/uuid"
)

// TestDecisionSpecFFI_BatchStateRoundtrip exercises the in-process
// equivalent of MageEncodeDecisionSpec + MageDecisionMaskNext: build
// batchSpecState rows directly from emitDecisionSpec output and feed
// them to nextMask. This tests the same Go code paths the cgo entry
// points use, avoiding the cgo-in-test awkwardness.
func TestDecisionSpecFFI_BatchStateRoundtrip(t *testing.T) {
	ids := testSpecIDs()
	scratch := newSpecOut()

	// Row 0: PRIORITY with 3 actions.
	pending0 := &apiPending{
		Kind:    "priority",
		Options: []apiOption{{Kind: "pass"}, {Kind: "play_land"}, {Kind: "cast_spell"}},
	}
	// Row 1: DECLARE_BLOCKERS, 2 blockers x 2 attackers.
	atkA := uuid.New()
	atkB := uuid.New()
	pending1 := &apiPending{
		Kind: "blockers",
		Options: []apiOption{
			{Kind: "blocker", ValidTargets: []apiTarget{{IDUUID: atkA}, {IDUUID: atkB}}},
			{Kind: "blocker", ValidTargets: []apiTarget{{IDUUID: atkA}}},
		},
	}

	state := &batchSpecState{rows: make([]rowSpecState, 2)}
	for i, pending := range []*apiPending{pending0, pending1} {
		scratch.tokens = scratch.tokens[:cap(scratch.tokens)]
		scratch.anchors = scratch.anchors[:cap(scratch.anchors)]
		emitDecisionSpec(pending, ids, scratch)
		row := &state.rows[i]
		row.decType = scratch.decisionType
		for k := int32(0); k < scratch.anchorsLen; k++ {
			switch scratch.anchors[k].kind {
			case anchorLegalAttacker:
				row.nLegalAttackers++
			case anchorLegalBlocker:
				row.nLegalBlockers++
			case anchorLegalTarget:
				row.nLegalTargets++
			case anchorLegalAction:
				row.nLegalActions++
			case anchorDefender:
				row.nDefenders++
			}
		}
		if scratch.decisionType == decTypeDeclareBlockers {
			row.legalEdgeBitmap = make([]byte, len(scratch.legalEdgeBitmap))
			copy(row.legalEdgeBitmap, scratch.legalEdgeBitmap)
		}
	}

	// Mask check: Row 0 PRIORITY at prefix len 0 must allow gPriorityOpen.
	const plMax = 8
	prefixTokens := make([]int32, 2*plMax)
	prefixPointers := make([]int32, 2*plMax)
	prefixLens := []int32{0, 0}
	v := int(grammarVocabSize)
	const na = 8
	vocabMask := make([]byte, 2*v)
	pointerMask := make([]byte, 2*na)

	for i := range 2 {
		row := &state.rows[i]
		in := decisionMaskInput{
			decType:         row.decType,
			nLegalAttackers: row.nLegalAttackers,
			nLegalBlockers:  row.nLegalBlockers,
			nLegalTargets:   row.nLegalTargets,
			nLegalActions:   row.nLegalActions,
			nDefenders:      row.nDefenders,
			maxValue:        row.maxValue,
			legalEdgeBitmap: row.legalEdgeBitmap,
			prefixTokens:    prefixTokens[i*plMax : i*plMax+int(prefixLens[i])],
			prefixPointers:  prefixPointers[i*plMax : i*plMax+int(prefixLens[i])],
			prefixLen:       prefixLens[i],
		}
		out := decisionMaskOutput{
			vocabMask:   vocabMask[i*v : (i+1)*v],
			pointerMask: pointerMask[i*na : (i+1)*na],
		}
		if err := nextMask(&in, &out); err != nil {
			t.Fatalf("row %d nextMask: %v", i, err)
		}
	}

	// Row 0 (PRIORITY) at prefix=0: vocab mask must have gPriorityOpen=1.
	if vocabMask[gPriorityOpen] != 1 {
		t.Fatalf("PRIORITY prefix=0 should allow gPriorityOpen; got vocab=%v", vocabMask[:v])
	}
	// Row 1 (DECLARE_BLOCKERS) at prefix=0: must have gDeclareBlockersOpn=1.
	if vocabMask[v+int(gDeclareBlockersOpn)] != 1 {
		t.Fatalf("DECLARE_BLOCKERS prefix=0 should allow open token; got vocab=%v", vocabMask[v:])
	}

	// Now advance row 0 to prefix=[gPriorityOpen] (1 token) — pointer mask
	// must select all 3 legal actions.
	prefixTokens[0] = gPriorityOpen
	prefixLens[0] = 1
	row := &state.rows[0]
	in := decisionMaskInput{
		decType:        row.decType,
		nLegalActions:  row.nLegalActions,
		prefixTokens:   prefixTokens[0:1],
		prefixPointers: prefixPointers[0:1],
		prefixLen:      1,
	}
	out := decisionMaskOutput{
		vocabMask:   vocabMask[0:v],
		pointerMask: pointerMask[0:na],
	}
	for i := range out.vocabMask {
		out.vocabMask[i] = 0
	}
	for i := range out.pointerMask {
		out.pointerMask[i] = 0
	}
	if err := nextMask(&in, &out); err != nil {
		t.Fatalf("PRIORITY prefix=1 nextMask: %v", err)
	}
	for i := range 3 {
		if out.pointerMask[i] != 1 {
			t.Fatalf("PRIORITY pointer[%d] should be 1; got %d", i, out.pointerMask[i])
		}
	}
	for i := 3; i < na; i++ {
		if out.pointerMask[i] != 0 {
			t.Fatalf("PRIORITY pointer[%d] should be 0; got %d", i, out.pointerMask[i])
		}
	}
}

// TestDecisionSpecFFI_HandleStore verifies the batchSpecState handle
// store: store and load roundtrip + release deletes.
func TestDecisionSpecFFI_HandleStore(t *testing.T) {
	state := &batchSpecState{rows: []rowSpecState{{decType: decTypePriority, nLegalActions: 5}}}
	id := storeBatchSpecState(state)
	if id == 0 {
		t.Fatalf("expected nonzero handle")
	}
	got := loadBatchSpecState(id)
	if got == nil || len(got.rows) != 1 || got.rows[0].nLegalActions != 5 {
		t.Fatalf("loaded state mismatch: %+v", got)
	}
	batchSpecStates.Delete(id)
	if loadBatchSpecState(id) != nil {
		t.Fatalf("expected handle to be deleted")
	}
}

// TestDecisionSpecTokenStore exercises the store roundtrip without
// going through cgo (which the C-side caller would do).
func TestDecisionSpecTokenStore_Setters(t *testing.T) {
	store := &decisionSpecTokenStore{
		ids: specTokenIDs{
			specOpen:      1,
			specClose:     2,
			decisionType:  3,
			legalAttacker: 4,
			legalBlocker:  5,
			legalTarget:   6,
			legalAction:   7,
			forAction:     8,
			maxValueOpen:  9,
			maxValueClose: 10,
			playerRef0:    11,
			playerRef1:    12,
		},
	}
	for i := range store.ids.dtName {
		store.ids.dtName[i] = int32(100 + i)
	}
	store.ids.stackRef = make([]int32, 16)
	for i := range store.ids.stackRef {
		store.ids.stackRef[i] = int32(200 + i)
	}
	decSpecTokensMu.Lock()
	prev := decSpecTokens
	decSpecTokens = store
	decSpecTokensMu.Unlock()
	defer func() {
		decSpecTokensMu.Lock()
		decSpecTokens = prev
		decSpecTokensMu.Unlock()
	}()
	got := getDecSpecTokens()
	if got == nil || got.ids.specOpen != 1 || got.ids.dtName[0] != 100 || got.ids.stackRef[15] != 215 {
		t.Fatalf("store roundtrip mismatch: %+v", got)
	}
}
