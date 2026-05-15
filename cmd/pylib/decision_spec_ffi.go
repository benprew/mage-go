package main

/*
#include <stdlib.h>
#include "abi.h"
*/
import "C"

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"
)

// decisionSpecTokenStore is the Go-side mirror of MageDecisionSpecTokens.
// All slices are borrowed from C; Python owns the underlying tensors.
type decisionSpecTokenStore struct {
	ids                  specTokenIDs
	maxValueDigits       []int32
	maxValueDigitOffsets []int32
	maxValueDigitMax     int32
}

var (
	decSpecTokensMu sync.RWMutex
	decSpecTokens   *decisionSpecTokenStore
)

func getDecSpecTokens() *decisionSpecTokenStore {
	decSpecTokensMu.RLock()
	defer decSpecTokensMu.RUnlock()
	return decSpecTokens
}

// MageRegisterDecisionSpecTokens registers the spec-tag ids + digit
// lookup table. Returns 0 on success, nonzero on a wire-format error.
//
//export MageRegisterDecisionSpecTokens
func MageRegisterDecisionSpecTokens(tables *C.MageDecisionSpecTokens) C.int32_t {
	defer func() { _ = recover() }()
	if tables == nil {
		decSpecTokensMu.Lock()
		decSpecTokens = nil
		decSpecTokensMu.Unlock()
		return 0
	}
	store := &decisionSpecTokenStore{
		ids: specTokenIDs{
			specOpen:      int32(tables.spec_open_id),
			specClose:     int32(tables.spec_close_id),
			decisionType:  int32(tables.decision_type_id),
			legalAttacker: int32(tables.legal_attacker_id),
			legalBlocker:  int32(tables.legal_blocker_id),
			legalTarget:   int32(tables.legal_target_id),
			legalAction:   int32(tables.legal_action_id),
			forAction:     int32(tables.for_action_id),
			maxValueOpen:  int32(tables.max_value_open_id),
			maxValueClose: int32(tables.max_value_close_id),
			playerRef0:    int32(tables.player_ref0_id),
			playerRef1:    int32(tables.player_ref1_id),
		},
		maxValueDigitMax: int32(tables.max_value_digit_max),
	}
	if tables.dt_name_ids != nil {
		dt := unsafe.Slice((*int32)(unsafe.Pointer(tables.dt_name_ids)), 7)
		copy(store.ids.dtName[:], dt)
	}
	if tables.stack_ref_ids != nil {
		store.ids.stackRef = unsafe.Slice((*int32)(unsafe.Pointer(tables.stack_ref_ids)), 16)
	}
	if store.maxValueDigitMax >= 0 && tables.max_value_digit_offsets != nil {
		offLen := int(store.maxValueDigitMax) + 2
		store.maxValueDigitOffsets = unsafe.Slice((*int32)(unsafe.Pointer(tables.max_value_digit_offsets)), offLen)
		if tables.max_value_digits != nil && len(store.maxValueDigitOffsets) > 0 {
			total := store.maxValueDigitOffsets[len(store.maxValueDigitOffsets)-1]
			if total > 0 {
				store.maxValueDigits = unsafe.Slice((*int32)(unsafe.Pointer(tables.max_value_digits)), int(total))
			}
		}
	}
	decSpecTokensMu.Lock()
	decSpecTokens = store
	decSpecTokensMu.Unlock()
	return 0
}

// batchSpecState captures per-env spec data from the most recent
// MageEncodeTokensPacked call. nextMask reads it via the batch handle.
type batchSpecState struct {
	rows []rowSpecState
}

type rowSpecState struct {
	decType         decisionType
	nLegalAttackers int32
	nLegalBlockers  int32
	nLegalTargets   int32
	nLegalActions   int32
	nDefenders      int32
	maxValue        int32
	legalEdgeBitmap []byte // copy, owned by Go
}

var (
	batchSpecStates  sync.Map // int64 -> *batchSpecState
	batchSpecCounter int64
)

func storeBatchSpecState(state *batchSpecState) int64 {
	id := atomic.AddInt64(&batchSpecCounter, 1)
	batchSpecStates.Store(id, state)
	return id
}

func loadBatchSpecState(id int64) *batchSpecState {
	v, ok := batchSpecStates.Load(id)
	if !ok {
		return nil
	}
	return v.(*batchSpecState)
}

//export MageReleaseBatchHandle
func MageReleaseBatchHandle(handle C.int64_t) {
	defer func() { _ = recover() }()
	batchSpecStates.Delete(int64(handle))
}

// MageEncodeDecisionSpec emits decision-spec tokens, anchors, and side
// tensors for each row in “req“. Returns a batch handle (via “handleOut“)
// that subsequent MageDecisionMaskNext calls can reference.
//
// This is a sibling of MageEncodeTokensPacked: it does not touch the
// inline-blank assembler outputs and does not need a token-budget. It
// assumes the caller has already invoked MageEncodeTokensPacked on the
// same batch and is supplying the per-row state-token lengths so that
// pointer_anchor_positions can be shifted into the combined stream.
//
//export MageEncodeDecisionSpec
func MageEncodeDecisionSpec(
	req *C.MageBatchRequest,
	stateTokenLensPtr *C.int32_t,
	specOut *C.MagePackedSpecOutputs,
	handleOut *C.int64_t,
) (res C.MageEncodeResult) {
	defer func() {
		if r := recover(); r != nil {
			res = newEncodeResult(0, mageEncodeErrEncodeFailure, fmt.Sprintf("panic: %v\n%s", r, debug.Stack()))
		}
	}()
	if req == nil || specOut == nil || handleOut == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req, spec_out, handle_out must be non-nil")
	}
	tokens := getDecSpecTokens()
	if tokens == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "MageRegisterDecisionSpecTokens must be called first")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	if n == 0 {
		*handleOut = 0
		return newEncodeResult(0, mageEncodeErrOK, "")
	}
	handles := unsafe.Slice((*int64)(unsafe.Pointer(req.handles)), n)
	var stateLens []int32
	if stateTokenLensPtr != nil {
		stateLens = unsafe.Slice((*int32)(unsafe.Pointer(stateTokenLensPtr)), n)
	}

	tSpec := int64(specOut.T_spec_max)
	nAnchors := int64(specOut.N_anchors_max)
	nBlk := int64(specOut.N_blockers_max)
	nAtk := int64(specOut.N_attackers_max)
	if tSpec <= 0 || nAnchors <= 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "T_spec_max and N_anchors_max must be positive")
	}

	specTokens := unsafe.Slice((*int32)(unsafe.Pointer(specOut.spec_tokens)), n*tSpec)
	specLens := unsafe.Slice((*int32)(unsafe.Pointer(specOut.spec_lens)), n)
	decTypes := unsafe.Slice((*int32)(unsafe.Pointer(specOut.decision_type)), n)
	apPos := unsafe.Slice((*int32)(unsafe.Pointer(specOut.pointer_anchor_positions)), n*nAnchors)
	apKinds := unsafe.Slice((*int32)(unsafe.Pointer(specOut.pointer_anchor_kinds)), n*nAnchors)
	apSubj := unsafe.Slice((*int32)(unsafe.Pointer(specOut.pointer_anchor_subjects)), n*nAnchors)
	apHand := unsafe.Slice((*int32)(unsafe.Pointer(specOut.pointer_anchor_handles)), n*nAnchors)
	apCnt := unsafe.Slice((*int32)(unsafe.Pointer(specOut.pointer_anchor_counts)), n)
	var bitmap []byte
	if specOut.legal_edge_bitmap != nil && nBlk > 0 && nAtk > 0 {
		bitmap = unsafe.Slice((*byte)(unsafe.Pointer(specOut.legal_edge_bitmap)), n*nBlk*nAtk)
	}
	bitmapNB := unsafe.Slice((*int32)(unsafe.Pointer(specOut.legal_edge_n_blockers)), n)
	bitmapNA := unsafe.Slice((*int32)(unsafe.Pointer(specOut.legal_edge_n_attackers)), n)

	// Reset entire output buffers to defaults.
	for i := range specTokens {
		specTokens[i] = 0
	}
	for i := range specLens {
		specLens[i] = 0
	}
	for i := range decTypes {
		decTypes[i] = -1
	}
	for i := range apPos {
		apPos[i] = -1
	}
	for i := range apKinds {
		apKinds[i] = -1
	}
	for i := range apSubj {
		apSubj[i] = 0
	}
	for i := range apHand {
		apHand[i] = -1
	}
	for i := range apCnt {
		apCnt[i] = 0
	}
	for i := range bitmap {
		bitmap[i] = 0
	}
	for i := range bitmapNB {
		bitmapNB[i] = 0
	}
	for i := range bitmapNA {
		bitmapNA[i] = 0
	}
	specOut.spec_overflow = 0

	state := &batchSpecState{rows: make([]rowSpecState, n)}

	scratch := specEmitterOut{
		tokens:               make([]int32, tSpec),
		anchors:              make([]pointerAnchor, nAnchors),
		maxValueDigits:       tokens.maxValueDigits,
		maxValueDigitOffsets: tokens.maxValueDigitOffsets,
		maxValueDigitMax:     tokens.maxValueDigitMax,
	}

	overflow := false

	for batchIdx := int64(0); batchIdx < n; batchIdx++ {
		h := getHandle(handles[batchIdx])
		var pending *apiPending
		if h != nil {
			h.mu.Lock()
			if !h.done {
				pending = buildPending(h.current)
			}
			h.mu.Unlock()
		}
		scratch.tokens = scratch.tokens[:tSpec]
		scratch.anchors = scratch.anchors[:nAnchors]
		scratch.legalEdgeBitmap = nil
		if pending == nil {
			scratch.reset()
			continue
		}
		if err := emitDecisionSpec(pending, &tokens.ids, &scratch); err != nil {
			continue
		}
		if scratch.decisionType == decTypeNone {
			continue
		}
		decTypes[batchIdx] = int32(scratch.decisionType)

		writeLen := scratch.tokensLen
		if int64(writeLen) > tSpec {
			writeLen = int32(tSpec)
			overflow = true
		}
		copy(specTokens[batchIdx*tSpec:batchIdx*tSpec+int64(writeLen)], scratch.tokens[:writeLen])
		specLens[batchIdx] = writeLen

		anchorLen := scratch.anchorsLen
		if int64(anchorLen) > nAnchors {
			anchorLen = int32(nAnchors)
			overflow = true
		}
		var stateLen int32
		if stateLens != nil {
			stateLen = stateLens[batchIdx]
		}
		for k := int32(0); k < anchorLen; k++ {
			a := scratch.anchors[k]
			off := batchIdx*nAnchors + int64(k)
			apPos[off] = a.tokenPosition + stateLen
			apKinds[off] = int32(a.kind)
			apSubj[off] = a.subjectIndex
			apHand[off] = a.handle
		}
		apCnt[batchIdx] = anchorLen

		row := &state.rows[batchIdx]
		row.decType = scratch.decisionType
		// Tally per-kind anchor counts.
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
		// max_value: capture for CHOOSE_MODE/CHOOSE_X.
		switch scratch.decisionType {
		case decTypeChooseMode:
			row.maxValue = int32(len(pending.Options))
		case decTypeChooseX:
			if pending.Amount > 0 {
				row.maxValue = int32(pending.Amount)
			} else if n := len(pending.Options); n > 0 {
				row.maxValue = int32(n - 1)
			}
		}
		// Side tensor: legal-edge bitmap for DECLARE_BLOCKERS.
		if scratch.decisionType == decTypeDeclareBlockers && scratch.nBlockers > 0 && scratch.nAttackers > 0 {
			nB := scratch.nBlockers
			nA := scratch.nAttackers
			bitmapNB[batchIdx] = nB
			bitmapNA[batchIdx] = nA
			// Capture into state (Go-owned copy) for nextMask use.
			row.legalEdgeBitmap = make([]byte, len(scratch.legalEdgeBitmap))
			copy(row.legalEdgeBitmap, scratch.legalEdgeBitmap)
			// Write into output buffer if shape fits.
			if bitmap != nil && int64(nB) <= nBlk && int64(nA) <= nAtk {
				rowBase := batchIdx * nBlk * nAtk
				for b := int32(0); b < nB; b++ {
					for a := int32(0); a < nA; a++ {
						bitmap[rowBase+int64(b)*nAtk+int64(a)] = scratch.legalEdgeBitmap[int(b)*int(nA)+int(a)]
					}
				}
			} else if bitmap != nil {
				overflow = true
			}
		}
	}

	if overflow {
		specOut.spec_overflow = 1
	}

	*handleOut = C.int64_t(storeBatchSpecState(state))
	return newEncodeResult(n, mageEncodeErrOK, "")
}

// MageDecisionMaskNext computes per-row vocab + pointer masks for one
// decoder step. Reads the per-row state captured by the most recent
// MageEncodeDecisionSpec call referenced by “batch_handle“.
//
//export MageDecisionMaskNext
func MageDecisionMaskNext(
	batchHandle C.int64_t,
	prefixTokensPtr *C.int32_t,
	prefixPointersPtr *C.int32_t,
	prefixLensPtr *C.int32_t,
	batchSize C.int32_t,
	prefixLenMax C.int32_t,
	grammarVocabSizeArg C.int32_t,
	nAnchorsMax C.int32_t,
	outVocabMaskPtr *C.uint8_t,
	outPointerMaskPtr *C.uint8_t,
) C.int32_t {
	defer func() { _ = recover() }()
	state := loadBatchSpecState(int64(batchHandle))
	if state == nil {
		return 1
	}
	bs := int(batchSize)
	if bs != len(state.rows) {
		return 2
	}
	plMax := int(prefixLenMax)
	v := int(grammarVocabSizeArg)
	na := int(nAnchorsMax)
	if v != int(grammarVocabSize) {
		return 3
	}
	prefixTokens := unsafe.Slice((*int32)(unsafe.Pointer(prefixTokensPtr)), bs*plMax)
	prefixPointers := unsafe.Slice((*int32)(unsafe.Pointer(prefixPointersPtr)), bs*plMax)
	prefixLens := unsafe.Slice((*int32)(unsafe.Pointer(prefixLensPtr)), bs)
	vocabMask := unsafe.Slice((*byte)(unsafe.Pointer(outVocabMaskPtr)), bs*v)
	pointerMask := unsafe.Slice((*byte)(unsafe.Pointer(outPointerMaskPtr)), bs*na)

	for i := range vocabMask {
		vocabMask[i] = 0
	}
	for i := range pointerMask {
		pointerMask[i] = 0
	}

	var wg sync.WaitGroup
	for i := 0; i < bs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			row := &state.rows[i]
			if row.decType == decTypeNone {
				return
			}
			pl := int(prefixLens[i])
			if pl < 0 {
				pl = 0
			}
			if pl > plMax {
				pl = plMax
			}
			in := decisionMaskInput{
				decType:         row.decType,
				nLegalAttackers: row.nLegalAttackers,
				nLegalBlockers:  row.nLegalBlockers,
				nLegalTargets:   row.nLegalTargets,
				nLegalActions:   row.nLegalActions,
				nDefenders:      row.nDefenders,
				maxValue:        row.maxValue,
				legalEdgeBitmap: row.legalEdgeBitmap,
				prefixTokens:    prefixTokens[i*plMax : i*plMax+pl],
				prefixPointers:  prefixPointers[i*plMax : i*plMax+pl],
				prefixLen:       int32(pl),
			}
			out := decisionMaskOutput{
				vocabMask:   vocabMask[i*v : (i+1)*v],
				pointerMask: pointerMask[i*na : (i+1)*na],
			}
			_ = nextMask(&in, &out)
		}(i)
	}
	wg.Wait()
	return 0
}

// MagePackCombinedTokens rewrites the packed token stream produced by
// MageEncodeTokensPacked so that each row's spec tokens (emitted into a side
// buffer by MageEncodeDecisionSpec) are concatenated immediately after that
// row's state tokens. After this call, “packed.token_ids[0:cu_seqlens[B]]“
// holds “state || spec“ for every row in row-major packed layout, with all
// supporting outputs (cu_seqlens, seq_lengths, state_positions,
// card_ref_positions, spec.pointer_anchor_positions) updated to the new
// combined-stream coordinates.
//
// The rewrite is in-place: state tokens are shifted forward (lower row index
// processed last) so the source slice is read before being overwritten by a
// later row's destination. Per-row card_ref_positions get the row's
// cumulative-spec offset added; per-row pointer_anchor_positions (which Go
// previously wrote in row-local combined coords) get the row's new packed
// start added so they share the card-ref convention.
//
// Returns 0 on success, -1 if the combined stream would exceed
// “token_capacity“ (caller must allocate “B * max_tokens“ large enough to
// hold state + spec for every row).
//
//export MagePackCombinedTokens
func MagePackCombinedTokens(
	n C.int32_t,
	tokenCapacity C.int32_t,
	maxCardRefs C.int32_t,
	packed *C.MagePackedTokenAssemblerOutputs,
	spec *C.MagePackedSpecOutputs,
) C.int32_t {
	defer func() { _ = recover() }()
	if packed == nil || spec == nil {
		return -2
	}
	nInt := int(n)
	if nInt == 0 {
		return 0
	}
	cardRefsW := int(maxCardRefs)
	tSpec := int(spec.T_spec_max)
	nAnchors := int(spec.N_anchors_max)

	tokens := unsafe.Slice((*int32)(unsafe.Pointer(packed.token_ids)), int(tokenCapacity))
	cu := unsafe.Slice((*int32)(unsafe.Pointer(packed.cu_seqlens)), nInt+1)
	seqLens := unsafe.Slice((*int32)(unsafe.Pointer(packed.seq_lengths)), nInt)
	statePos := unsafe.Slice((*int32)(unsafe.Pointer(packed.state_positions)), nInt)
	var cardRefs []int32
	if cardRefsW > 0 {
		cardRefs = unsafe.Slice((*int32)(unsafe.Pointer(packed.card_ref_positions)), nInt*cardRefsW)
	}
	var specTokens []int32
	if tSpec > 0 {
		specTokens = unsafe.Slice((*int32)(unsafe.Pointer(spec.spec_tokens)), nInt*tSpec)
	}
	specLens := unsafe.Slice((*int32)(unsafe.Pointer(spec.spec_lens)), nInt)
	var anchorPos []int32
	if nAnchors > 0 {
		anchorPos = unsafe.Slice((*int32)(unsafe.Pointer(spec.pointer_anchor_positions)), nInt*nAnchors)
	}

	// Compute new combined cu_seqlens and check overflow before mutating.
	newCu := make([]int32, nInt+1)
	for b := 0; b < nInt; b++ {
		newCu[b+1] = newCu[b] + seqLens[b] + specLens[b]
	}
	if int(newCu[nInt]) > int(tokenCapacity) {
		return -1
	}

	// Walk rows in reverse so the destination range never clobbers a yet-
	// unread source range (every row's new start is >= its old start).
	for b := nInt - 1; b >= 0; b-- {
		oldStart := statePos[b]
		newStart := newCu[b]
		sLen := seqLens[b]
		pLen := specLens[b]
		// Move state tokens from [oldStart, oldStart+sLen) to
		// [newStart, newStart+sLen). Copy right-to-left within the row to
		// avoid overwriting the source when newStart > oldStart.
		if oldStart != newStart {
			for i := sLen - 1; i >= 0; i-- {
				tokens[newStart+i] = tokens[oldStart+i]
			}
		}
		// Append spec tokens immediately after state.
		for i := int32(0); i < pLen; i++ {
			tokens[newStart+sLen+i] = specTokens[int32(b)*int32(tSpec)+i]
		}
		// Shift card_ref_positions for this row by (newStart - oldStart),
		// preserving -1 sentinels.
		delta := newStart - oldStart
		if delta != 0 && cardRefsW > 0 {
			base := b * cardRefsW
			for i := 0; i < cardRefsW; i++ {
				v := cardRefs[base+i]
				if v >= 0 {
					cardRefs[base+i] = v + delta
				}
			}
		}
		// Shift pointer_anchor_positions: row-local combined → packed
		// combined (add the row's new packed start).
		if nAnchors > 0 {
			base := b * nAnchors
			for i := 0; i < nAnchors; i++ {
				v := anchorPos[base+i]
				if v >= 0 {
					anchorPos[base+i] = v + newStart
				}
			}
		}
		statePos[b] = newStart
		seqLens[b] = sLen + pLen
	}
	for b := 0; b <= nInt; b++ {
		cu[b] = newCu[b]
	}
	return 0
}
