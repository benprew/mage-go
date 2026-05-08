package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// scratchPool keys a free list of encodeScratch instances by output-buffer
// pointer, alongside the per-row directDirty state for that buffer. Each
// parallel worker takes its own scratch (see “acquireScratch“) so its
// emitter / out / cardIDToSlot don't race; the directDirty state, however,
// must be 1:1 with the OUTPUT BUFFER, not the scratch — otherwise a row
// handled by scratch A in one call and scratch B in the next has a stale
// dirty record on whichever scratch is reused later, and reset's partial
// clear leaves residue from the OTHER scratch's writes in the buffer. See
// the per-row directDirty comment on “directDirtyState“.
type scratchPool struct {
	mu          sync.Mutex
	available   []*encodeScratch
	directDirty []directDirtyState
}

// ensureDirty grows “p.directDirty“ to at least “n“ entries. Caller
// must hold p.mu OR be the only writer (e.g. before launching workers).
func (p *scratchPool) ensureDirty(n int) {
	if cap(p.directDirty) < n {
		grown := make([]directDirtyState, n)
		copy(grown, p.directDirty)
		p.directDirty = grown
	} else if len(p.directDirty) < n {
		p.directDirty = p.directDirty[:n]
	}
}

// rowDirty returns the per-row dirty entry for “batchIdx“. Concurrent
// callers from different workers are safe as long as each batchIdx is
// owned by at most one worker — the pointer aliases the slice element,
// and slice growth is done up-front via ensureDirty.
func (p *scratchPool) rowDirty(batchIdx int64) *directDirtyState {
	return &p.directDirty[batchIdx]
}

var scratchPools sync.Map // uintptr -> *scratchPool

// scratchPoolKey returns a stable pool key for the given output views, or
// 0 if no packed buffer is bound (in which case caching is skipped). Uses
// the address of packedTokenIDs[0] since that buffer is allocated once
// per outputs object and reused on every call.
func scratchPoolKey(views outputViews) uintptr {
	if len(views.packedTokenIDs) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&views.packedTokenIDs[0]))
}

// scratchPoolFor returns the per-buffer scratch pool, lazily creating it.
// Returns nil when “key == 0“ (no buffer bound — caller must supply a
// fresh “directDirtyState“ for each fillTokenAssemblyDirectPacked call).
func scratchPoolFor(key uintptr) *scratchPool {
	if key == 0 {
		return nil
	}
	val, ok := scratchPools.Load(key)
	if !ok {
		val, _ = scratchPools.LoadOrStore(key, &scratchPool{})
	}
	return val.(*scratchPool)
}

func acquireScratch(key uintptr) *encodeScratch {
	p := scratchPoolFor(key)
	if p == nil {
		return newEncodeScratch()
	}
	p.mu.Lock()
	var s *encodeScratch
	if n := len(p.available); n > 0 {
		s = p.available[n-1]
		p.available[n-1] = nil
		p.available = p.available[:n-1]
	}
	p.mu.Unlock()
	if s == nil {
		s = newEncodeScratch()
	}
	return s
}

func releaseScratch(key uintptr, s *encodeScratch) {
	if key == 0 {
		return
	}
	val, ok := scratchPools.Load(key)
	if !ok {
		return
	}
	p := val.(*scratchPool)
	p.mu.Lock()
	p.available = append(p.available, s)
	p.mu.Unlock()
}

const (
	zoneSlotCount                = 50
	gameInfoDim                  = 90
	optionScalarDim              = 14
	targetScalarDim              = 2
	maxCardsPerZone              = 10
	maxLife                      = 40.0
	maxTurn                      = 20.0
	maxMana                      = 10.0
	maxLibrary                   = 60.0
	maxPendingOpt                = 20.0
	maxAbilityIndex              = 8.0
	maxAmount                    = 20.0
	maxTargetOverflow            = 32.0
	unknownTargetID              = 3
	mageEncodeErrOK              = 0
	mageEncodeErrArg             = 1
	mageEncodeErrHandle          = 2
	mageEncodeErrPlayer          = 3
	mageEncodeErrOver            = 4
	mageEncodeErrBuffer          = 5
	mageEncodeErrEncode          = 6
	mageEncodeErrInvalidArgument = mageEncodeErrArg
	mageEncodeErrUnknownHandle   = mageEncodeErrHandle
	mageEncodeErrInvalidPlayer   = mageEncodeErrPlayer
	mageEncodeErrGameOver        = mageEncodeErrOver
	mageEncodeErrBufferTooSmall  = mageEncodeErrBuffer
	mageEncodeErrEncodeFailure   = mageEncodeErrEncode
	packedParallelMinRows        = 128
	packedParallelMaxWorkers     = 8
)

var (
	manaSymbols           = [...]string{"W", "U", "B", "R", "G", "C"}
	stepNames             = [...]string{"Untap", "Upkeep", "Draw", "Precombat Main", "Begin Combat", "Declare Attackers", "Declare Blockers", "Combat Damage", "End Combat", "Postcombat Main", "End", "Cleanup", "Unknown"}
	pendingKinds          = [...]string{"priority", "attackers", "blockers", "permanent", "cards_from_hand", "mana_color", "card_from_library", "may", "mode", "number", "unknown"}
	actionKinds           = [...]string{"pass", "play_land", "cast_spell", "activate_ability", "attacker", "blocker", "choice", "unknown"}
	traceKinds            = [...]string{"priority", "attackers", "blockers", "choice_index", "choice_ids", "choice_color", "may"}
	zoneSpecs             = [...]zoneSpec{{zone: "hand", owner: "self"}, {zone: "graveyard", owner: "self"}, {zone: "graveyard", owner: "opponent"}, {zone: "battlefield", owner: "self"}, {zone: "battlefield", owner: "opponent"}}
	stepNamesNorm         = normalizedKeys(stepNames[:])
	pendingKindsNorm      = normalizedKeys(pendingKinds[:])
	actionKindsNorm       = normalizedKeys(actionKinds[:])
	traceKindsNorm        = normalizedKeys(traceKinds[:])
	cardRowsOnce          sync.Once
	cardRowByName         map[string]int64
	cardRowByRawName      map[string]int64
	cardRowOverrideMu     sync.RWMutex
	cardRowOverrides      = map[string]int64{}
	cardRowOverridesByRaw = map[string]int64{}
	cardRowsOverridden    bool
)

type zoneSpec struct {
	zone  string
	owner string
}

type encodeError struct {
	code    int64
	message string
}

type encodeConfig struct {
	maxOptions          int64
	maxTargetsPerOption int64
	maxCachedChoices    int64
	zoneSlotCount       int64
	gameInfoDim         int64
	optionScalarDim     int64
	targetScalarDim     int64
	decisionCapacity    int64
	emitRenderPlan      bool
	renderPlanCapacity  int64
	// dedupCardBodies turns on the v2 ``<dict>`` opcode set: each unique
	// card cache row in the snapshot is spliced once at the top, and per-zone
	// occurrences become short ``<card-ref>``-anchored references back to
	// the dict entry instead of full body splices. Off by default — the
	// native token assembler does not yet understand the v2 opcodes.
	dedupCardBodies  bool
	tokenMaxTokens   int32
	tokenMaxOptions  int32
	tokenMaxTargets  int32
	tokenMaxCardRefs int32
	blankMaxBlanks   int32
	blankMaxLegal    int32
	// emitTokensPacked turns on the native packed token-assembler pass after
	// render-plan emission. Output buffers live in outputViews.
	emitTokensPacked bool
}

type outputViews struct {
	traceKindID        []int64
	slotCardRows       []int64
	slotOccupied       []float32
	slotTapped         []float32
	gameInfo           []float32
	pendingKindID      []int64
	numPresentOptions  []int64
	optionKindIDs      []int64
	optionScalars      []float32
	optionMask         []float32
	optionRefSlotIdx   []int64
	optionRefCardRow   []int64
	targetMask         []float32
	targetTypeIDs      []int64
	targetScalars      []float32
	targetOverflow     []float32
	targetRefSlotIdx   []int64
	targetRefIsPlayer  []byte
	targetRefIsSelf    []byte
	mayMask            []byte
	decisionStart      []int64
	decisionCount      []int64
	decisionOptionIdx  []int64
	decisionTargetIdx  []int64
	decisionMask       []byte
	usesNoneHead       []byte
	renderPlan         []int32
	renderPlanLengths  []int64
	renderPlanOverflow []int64

	// Packed (varlen) token-assembler outputs. ``packedTokenIDs`` is sized
	// [B*max_tokens]; per-token seq/position metadata is derived by Python.
	packedTokenIDs        []int32
	packedCuSeqlens       []int32 // [B+1]
	packedSeqLengths      []int32 // [B]
	packedStatePositions  []int32 // [B]
	packedCardRefPos      []int32
	packedTokenOverflow   []int32
	packedBlankPos        []int32
	packedBlankKind       []int32
	packedBlankGroup      []int32
	packedBlankGroupKind  []int32
	packedBlankOptionIdx  []int32
	packedBlankLegalIDs   []int32
	packedBlankLegalMask  []byte
	packedBlankOverflow   []int32
	packedBlankCount      []int32
	packedBlankLegalCount []int32
}

type batchRequest struct {
	handles      []int64
	perspectives []int64
}

type stateCard struct {
	id     string
	name   string
	tapped bool
}

func validateEncodeConfig(cfg encodeConfig) *encodeError {
	switch {
	case cfg.maxOptions < 0 || cfg.maxTargetsPerOption < 0 || cfg.maxCachedChoices < 0 || cfg.decisionCapacity < 0:
		return &encodeError{code: mageEncodeErrArg, message: "config sizes must be non-negative"}
	case cfg.zoneSlotCount != zoneSlotCount:
		return &encodeError{code: mageEncodeErrArg, message: fmt.Sprintf("zone_slot_count=%d, want %d", cfg.zoneSlotCount, zoneSlotCount)}
	case cfg.gameInfoDim != gameInfoDim:
		return &encodeError{code: mageEncodeErrArg, message: fmt.Sprintf("game_info_dim=%d, want %d", cfg.gameInfoDim, gameInfoDim)}
	case cfg.optionScalarDim != optionScalarDim:
		return &encodeError{code: mageEncodeErrArg, message: fmt.Sprintf("option_scalar_dim=%d, want %d", cfg.optionScalarDim, optionScalarDim)}
	case cfg.targetScalarDim != targetScalarDim:
		return &encodeError{code: mageEncodeErrArg, message: fmt.Sprintf("target_scalar_dim=%d, want %d", cfg.targetScalarDim, targetScalarDim)}
	case cfg.maxCachedChoices < cfg.maxOptions:
		return &encodeError{code: mageEncodeErrArg, message: "max_cached_choices must be >= max_options"}
	case cfg.maxCachedChoices < cfg.maxTargetsPerOption+1:
		return &encodeError{code: mageEncodeErrArg, message: "max_cached_choices must be >= max_targets_per_option + 1"}
	case (cfg.emitRenderPlan || cfg.emitTokensPacked) && cfg.renderPlanCapacity <= 0:
		return &encodeError{code: mageEncodeErrArg, message: "render_plan_capacity must be positive when render-plan-backed token assembly is set"}
	case cfg.renderPlanCapacity > math.MaxInt32:
		return &encodeError{code: mageEncodeErrArg, message: "render_plan_capacity must fit in int32"}
	}
	return nil
}

func encodeBatchGo(req batchRequest, cfg encodeConfig, views outputViews) (int64, *encodeError) {
	if cfg.emitTokensPacked && !cfg.emitRenderPlan && len(req.handles) >= packedParallelMinRows {
		return encodeBatchGoPackedParallel(req, cfg, views)
	}

	callStart := time.Time{}
	var gameTiming, renderTiming, assemblyTiming, metadataTiming time.Duration
	if cfg.emitTokensPacked {
		callStart = time.Now()
	}
	clearOutputViews(views, cfg)
	decisionCursor := int64(0)
	// Running write cursor into the packed token buffer. Only advanced
	// when emitTokensPacked is set; ignored otherwise.
	packedCursor := int32(0)
	if cfg.emitTokensPacked && len(views.packedCuSeqlens) > 0 {
		views.packedCuSeqlens[0] = 0
	}
	poolKey := scratchPoolKey(views)
	scratch := acquireScratch(poolKey)
	defer releaseScratch(poolKey, scratch)
	pool := scratchPoolFor(poolKey)
	// Per-row directDirty lives on the buffer-keyed pool (not on scratch),
	// so all scratches that touch this buffer share the same dirty record
	// and partial-clears reflect what's actually in the buffer.
	var fallbackDirty directDirtyState
	if pool != nil {
		pool.mu.Lock()
		pool.ensureDirty(len(req.handles))
		pool.mu.Unlock()
	}
	rowDirty := func(batchIdx int64) *directDirtyState {
		if pool != nil {
			return pool.rowDirty(batchIdx)
		}
		// No buffer bound — fall back to a fresh state per call. Forces
		// full clears, which is correct (no buffer to track).
		fallbackDirty = directDirtyState{}
		return &fallbackDirty
	}
	for batchIdx, handleID := range req.handles {
		h := getHandle(handleID)
		if h == nil {
			return decisionCursor, &encodeError{code: mageEncodeErrHandle, message: fmt.Sprintf("unknown handle %d", handleID)}
		}

		h.mu.Lock()
		if h.done {
			h.mu.Unlock()
			return decisionCursor, &encodeError{code: mageEncodeErrOver, message: fmt.Sprintf("handle %d is over", handleID)}
		}
		state := cachedSnapshotState(h)
		pending := buildPending(h.current)
		if pending == nil {
			h.mu.Unlock()
			return decisionCursor, &encodeError{code: mageEncodeErrEncode, message: fmt.Sprintf("handle %d has no pending request", handleID)}
		}

		requestedPerspective := int64(-1)
		if req.perspectives != nil {
			requestedPerspective = req.perspectives[batchIdx]
		}
		playerIdx, err := resolvePerspectivePlayerIndex(state, pending, requestedPerspective)
		if err != nil {
			h.mu.Unlock()
			return decisionCursor, err
		}

		scratch.reset()
		cardIDToSlot := scratch.cardIDToSlot
		phaseStart := time.Now()
		if err := fillStateEncoding(int64(batchIdx), state, pending, playerIdx, cfg, views, cardIDToSlot); err != nil {
			h.mu.Unlock()
			return decisionCursor, err
		}
		if err := fillActionEncoding(int64(batchIdx), state, pending, playerIdx, cfg, views, cardIDToSlot); err != nil {
			h.mu.Unlock()
			return decisionCursor, err
		}
		if cfg.emitTokensPacked {
			gameTiming += time.Since(phaseStart)
		}
		if cfg.emitRenderPlan {
			renderBatchIdx := int64(batchIdx)
			renderViews := views
			phaseStart = time.Now()
			if err := fillRenderPlan(renderBatchIdx, state, pending, playerIdx, cfg, renderViews, scratch); err != nil {
				h.mu.Unlock()
				return decisionCursor, err
			}
			if cfg.emitTokensPacked {
				renderTiming += time.Since(phaseStart)
			}
		}
		if cfg.emitTokensPacked {
			phaseStart = time.Now()
			var advanced int32
			var metadata time.Duration
			var err *encodeError
			if cfg.emitRenderPlan {
				advanced, metadata, err = fillTokenAssemblyPacked(
					int64(batchIdx),
					int64(batchIdx),
					packedCursor,
					cfg,
					views,
					views,
					scratch,
				)
			} else {
				advanced, metadata, err = fillTokenAssemblyDirectPacked(
					int64(batchIdx),
					packedCursor,
					state,
					pending,
					playerIdx,
					cfg,
					views,
					scratch,
					rowDirty(int64(batchIdx)),
				)
			}
			if err != nil {
				h.mu.Unlock()
				return decisionCursor, err
			}
			elapsed := time.Since(phaseStart)
			assemblyTiming += elapsed - metadata
			metadataTiming += metadata
			packedCursor = advanced
		}
		phaseStart = time.Now()
		written, err := fillDecisionEncoding(int64(batchIdx), pending, cfg, views, decisionCursor)
		if cfg.emitTokensPacked {
			gameTiming += time.Since(phaseStart)
		}
		h.mu.Unlock()
		if err != nil {
			return decisionCursor, err
		}
		decisionCursor += written
	}
	if cfg.emitTokensPacked {
		addPackedEncodeTiming(
			time.Since(callStart),
			gameTiming,
			renderTiming,
			assemblyTiming,
			metadataTiming,
		)
	}
	return decisionCursor, nil
}

func encodeBatchGoPackedParallel(req batchRequest, cfg encodeConfig, views outputViews) (int64, *encodeError) {
	callStart := time.Now()
	clearOutputViews(views, cfg)
	if len(views.packedCuSeqlens) > 0 {
		views.packedCuSeqlens[0] = 0
	}

	n := len(req.handles)
	decisionRows := make([]int64, n)
	gameTimings := make([]time.Duration, n)
	renderTimings := make([]time.Duration, n)
	assemblyTimings := make([]time.Duration, n)
	metadataTimings := make([]time.Duration, n)
	pendings := make([]*apiPending, n)

	workers := packedEncodeWorkerCount(n)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr *encodeError
	setErr := func(err *encodeError) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}

	poolKey := scratchPoolKey(views)
	pool := scratchPoolFor(poolKey)
	if pool != nil {
		// Size the per-buffer dirty slice once up-front so workers can
		// take aliasing pointers into it without locking — each worker
		// owns a disjoint row range.
		pool.mu.Lock()
		pool.ensureDirty(n)
		pool.mu.Unlock()
	}
	for workerIdx := range workers {
		start := workerIdx * n / workers
		end := (workerIdx + 1) * n / workers
		if start == end {
			continue
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			scratch := acquireScratch(poolKey)
			defer releaseScratch(poolKey, scratch)
			for batchIdx := start; batchIdx < end; batchIdx++ {
				errMu.Lock()
				stopped := firstErr != nil
				errMu.Unlock()
				if stopped {
					return
				}
				handleID := req.handles[batchIdx]
				h := getHandle(handleID)
				if h == nil {
					setErr(&encodeError{code: mageEncodeErrHandle, message: fmt.Sprintf("unknown handle %d", handleID)})
					return
				}

				h.mu.Lock()
				if h.done {
					h.mu.Unlock()
					setErr(&encodeError{code: mageEncodeErrOver, message: fmt.Sprintf("handle %d is over", handleID)})
					return
				}
				state := cachedSnapshotState(h)
				pending := buildPending(h.current)
				if pending == nil {
					h.mu.Unlock()
					setErr(&encodeError{code: mageEncodeErrEncode, message: fmt.Sprintf("handle %d has no pending request", handleID)})
					return
				}
				pendings[batchIdx] = pending

				requestedPerspective := int64(-1)
				if req.perspectives != nil {
					requestedPerspective = req.perspectives[batchIdx]
				}
				playerIdx, encErr := resolvePerspectivePlayerIndex(state, pending, requestedPerspective)
				if encErr != nil {
					h.mu.Unlock()
					setErr(encErr)
					return
				}

				scratch.reset()
				cardIDToSlot := scratch.cardIDToSlot
				phaseStart := time.Now()
				if encErr := fillStateEncoding(int64(batchIdx), state, pending, playerIdx, cfg, views, cardIDToSlot); encErr != nil {
					h.mu.Unlock()
					setErr(encErr)
					return
				}
				if encErr := fillActionEncoding(int64(batchIdx), state, pending, playerIdx, cfg, views, cardIDToSlot); encErr != nil {
					h.mu.Unlock()
					setErr(encErr)
					return
				}
				gameTimings[batchIdx] += time.Since(phaseStart)

				rowStart := int32(int64(batchIdx) * int64(cfg.tokenMaxTokens))
				phaseStart = time.Now()
				var dirty *directDirtyState
				if pool != nil {
					dirty = pool.rowDirty(int64(batchIdx))
				} else {
					dirty = &directDirtyState{}
				}
				_, metadata, encErr := fillTokenAssemblyDirectPacked(
					int64(batchIdx),
					rowStart,
					state,
					pending,
					playerIdx,
					cfg,
					views,
					scratch,
					dirty,
				)
				if encErr != nil {
					h.mu.Unlock()
					setErr(encErr)
					return
				}
				elapsed := time.Since(phaseStart)
				assemblyTimings[batchIdx] += elapsed - metadata
				metadataTimings[batchIdx] += metadata

				phaseStart = time.Now()
				decisionRows[batchIdx] = decisionRowsForPending(pending, cfg)
				gameTimings[batchIdx] += time.Since(phaseStart)
				h.mu.Unlock()
			}
		}(start, end)
	}
	wg.Wait()
	if firstErr != nil {
		return 0, firstErr
	}

	decisionCursor := int64(0)
	for batchIdx, count := range decisionRows {
		if decisionCursor+count > cfg.decisionCapacity {
			return decisionCursor, &encodeError{code: mageEncodeErrBuffer, message: "decision_capacity too small for decision rows"}
		}
		if pending := pendings[batchIdx]; pending != nil {
			written, encErr := fillDecisionEncoding(int64(batchIdx), pending, cfg, views, decisionCursor)
			if encErr != nil {
				return decisionCursor, encErr
			}
			if written != count {
				return decisionCursor, &encodeError{
					code:    mageEncodeErrEncode,
					message: fmt.Sprintf("decision row count mismatch for batch row %d: precomputed=%d written=%d", batchIdx, count, written),
				}
			}
		}
		decisionCursor += count
	}

	metadataStart := time.Now()
	if encErr := compactPackedRows(views, cfg); encErr != nil {
		return decisionCursor, encErr
	}
	metadataTimings[0] += time.Since(metadataStart)

	var gameTiming, renderTiming, assemblyTiming, metadataTiming time.Duration
	for i := range n {
		gameTiming += gameTimings[i]
		renderTiming += renderTimings[i]
		assemblyTiming += assemblyTimings[i]
		metadataTiming += metadataTimings[i]
	}
	addPackedEncodeTiming(
		time.Since(callStart),
		gameTiming,
		renderTiming,
		assemblyTiming,
		metadataTiming,
	)
	return decisionCursor, nil
}

func packedEncodeWorkerCount(n int) int {
	workers := minInt(n, runtime.GOMAXPROCS(0))
	workers = minInt(workers, packedParallelMaxWorkers)
	if raw := os.Getenv("MAGE_PACKED_ENCODE_WORKERS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			workers = minInt(n, parsed)
		}
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}

func compactPackedRows(views outputViews, cfg encodeConfig) *encodeError {
	n := len(views.packedSeqLengths)
	packedCursor := int32(0)
	for batchIdx := range n {
		length := views.packedSeqLengths[batchIdx]
		if length < 0 || length > cfg.tokenMaxTokens {
			return &encodeError{code: mageEncodeErrEncode, message: fmt.Sprintf("invalid packed sequence length %d at batch row %d", length, batchIdx)}
		}
		views.packedCuSeqlens[batchIdx+1] = packedCursor + length
		rowStart := int32(int64(batchIdx) * int64(cfg.tokenMaxTokens))
		if length > 0 && packedCursor != rowStart {
			copy(
				views.packedTokenIDs[packedCursor:packedCursor+length],
				views.packedTokenIDs[rowStart:rowStart+length],
			)
		}
		delta := packedCursor - rowStart
		if delta != 0 {
			rebasePackedPositions(views, cfg, int64(batchIdx), delta)
		}
		views.packedStatePositions[batchIdx] = packedCursor
		packedCursor += length
	}
	return nil
}

func clearOutputViews(view outputViews, cfg encodeConfig) {
	fillInt64(view.traceKindID, 0)
	fillInt64(view.slotCardRows, 0)
	fillFloat32(view.slotOccupied, 0)
	fillFloat32(view.slotTapped, 0)
	fillFloat32(view.gameInfo, 0)
	fillInt64(view.pendingKindID, 0)
	fillInt64(view.numPresentOptions, 0)
	fillInt64(view.optionKindIDs, 0)
	fillFloat32(view.optionScalars, 0)
	fillFloat32(view.optionMask, 0)
	fillInt64(view.optionRefSlotIdx, -1)
	fillInt64(view.optionRefCardRow, -1)
	fillFloat32(view.targetMask, 0)
	fillInt64(view.targetTypeIDs, unknownTargetID)
	fillFloat32(view.targetScalars, 0)
	fillFloat32(view.targetOverflow, 0)
	fillInt64(view.targetRefSlotIdx, -1)
	fillBytes(view.targetRefIsPlayer, 0)
	fillBytes(view.targetRefIsSelf, 0)
	fillBytes(view.mayMask, 0)
	fillInt64(view.decisionStart, 0)
	fillInt64(view.decisionCount, 0)
	fillInt64(view.decisionOptionIdx, -1)
	fillInt64(view.decisionTargetIdx, -1)
	fillBytes(view.decisionMask, 0)
	fillBytes(view.usesNoneHead, 0)
	fillInt32(view.renderPlan, 0)
	fillInt64(view.renderPlanLengths, 0)
	fillInt64(view.renderPlanOverflow, 0)
	if cfg.emitTokensPacked {
		// Packed token outputs are reset by the Python wrapper before
		// every reuse. Avoid clearing these large slabs a second time here;
		// the assembler only writes the live token region and active anchors.
	}
}

// fillTokenAssemblyPacked writes one row's worth of tokens into the
// shared packed output buffer starting at “packedCursor“. Returns the
// new cursor (one past the last live token) so the caller can chain
// rows without an outer-loop allocation. Anchors are written as
// absolute offsets into the packed buffer.
func fillTokenAssemblyPacked(
	planBatchIdx int64,
	outputBatchIdx int64,
	packedCursor int32,
	cfg encodeConfig,
	planView outputViews,
	outputView outputViews,
	scratch *encodeScratch,
) (int32, time.Duration, *encodeError) {
	tables := getTokenTables()
	if tables == nil {
		return packedCursor, 0, &encodeError{
			code:    mageEncodeErrEncodeFailure,
			message: "MageRegisterTokenTables must be called before MageEncodeTokensPacked",
		}
	}
	planStart := planBatchIdx * cfg.renderPlanCapacity
	planLen := planView.renderPlanLengths[planBatchIdx]
	plan := planView.renderPlan[planStart : planStart+planLen]

	mt := int64(cfg.tokenMaxTokens)
	mo := int64(cfg.tokenMaxOptions)
	mtg := int64(cfg.tokenMaxTargets)
	mcr := int64(cfg.tokenMaxCardRefs)
	mb := int64(cfg.blankMaxBlanks)
	mv := int64(cfg.blankMaxLegal)

	// Carve a row-sized scratch slice straight out of the packed buffer
	// at the running cursor. The assembler writes tokens into this view
	// using its own 0-based local cursor; with cursorBase=packedCursor
	// the anchor positions land as absolute offsets.
	rowStart := int64(packedCursor)
	rowEnd := rowStart + mt
	if rowEnd > int64(len(outputView.packedTokenIDs)) {
		return packedCursor, 0, &encodeError{
			code:    mageEncodeErrInvalidArgument,
			message: "packed token buffer too small (need >= B*max_tokens)",
		}
	}

	out := &tokenAssemblerOut{
		tokenIDs:    outputView.packedTokenIDs[rowStart:rowEnd],
		cardRefPos:  outputView.packedCardRefPos[outputBatchIdx*mcr : (outputBatchIdx+1)*mcr],
		maxOptions:  cfg.tokenMaxOptions,
		maxTargets:  cfg.tokenMaxTargets,
		maxCardRefs: cfg.tokenMaxCardRefs,
		cursorBase:  packedCursor,
	}
	out.optionPos, out.optionMask, out.targetPos, out.targetMask = scratch.packedAnchorScratch(
		mo,
		mo*mtg,
	)
	if mb > 0 && mv > 0 && len(outputView.packedBlankPos) > 0 {
		rowBlankStart := outputBatchIdx * mb
		rowBlankEnd := rowBlankStart + mb
		rowLegalStart := outputBatchIdx * mb * mv
		rowLegalEnd := rowLegalStart + mb*mv
		collector := &scratch.blankCollector
		collector.positions = outputView.packedBlankPos[rowBlankStart:rowBlankEnd]
		collector.kind = outputView.packedBlankKind[rowBlankStart:rowBlankEnd]
		collector.group = outputView.packedBlankGroup[rowBlankStart:rowBlankEnd]
		collector.groupKind = outputView.packedBlankGroupKind[rowBlankStart:rowBlankEnd]
		collector.optionIdx = outputView.packedBlankOptionIdx[rowBlankStart:rowBlankEnd]
		collector.legalIDs = outputView.packedBlankLegalIDs[rowLegalStart:rowLegalEnd]
		collector.legalMask = outputView.packedBlankLegalMask[rowLegalStart:rowLegalEnd]
		if outputBatchIdx >= 0 && outputBatchIdx < int64(len(outputView.packedBlankCount)) {
			collector.count = &outputView.packedBlankCount[outputBatchIdx]
		} else {
			collector.count = nil
		}
		collector.legalCount = outputView.packedBlankLegalCount[rowBlankStart:rowBlankEnd]
		if outputBatchIdx >= 0 && outputBatchIdx < int64(len(outputView.packedBlankOverflow)) {
			collector.overflow = &outputView.packedBlankOverflow[outputBatchIdx]
			outputView.packedBlankOverflow[outputBatchIdx] = 0
		} else {
			collector.overflow = nil
		}
		collector.reset(cfg.blankMaxBlanks, cfg.blankMaxLegal)
		out.blank = collector
	}

	outputView.packedSeqLengths[outputBatchIdx] = 0
	outputView.packedStatePositions[outputBatchIdx] = 0
	outputView.packedCuSeqlens[outputBatchIdx+1] = packedCursor
	outputView.packedTokenOverflow[outputBatchIdx] = 0

	cursor, overflow, err := assembleTokensFromPlan(plan, tables, out, cfg.tokenMaxTokens)
	if err != nil {
		return packedCursor, 0, &encodeError{
			code:    mageEncodeErrEncodeFailure,
			message: err.Error(),
		}
	}

	metadataStart := time.Now()
	outputView.packedSeqLengths[outputBatchIdx] = cursor
	outputView.packedStatePositions[outputBatchIdx] = packedCursor
	outputView.packedCuSeqlens[outputBatchIdx+1] = packedCursor + cursor
	if overflow {
		outputView.packedTokenOverflow[outputBatchIdx] = 1
	}
	return packedCursor + cursor, time.Since(metadataStart), nil
}

func rebasePackedPositions(view outputViews, cfg encodeConfig, batchIdx int64, delta int32) {
	mcr := cfg.tokenMaxCardRefs

	if batchIdx >= 0 && batchIdx < int64(len(view.packedStatePositions)) {
		view.packedStatePositions[batchIdx] += delta
	}

	cardStart := batchIdx * int64(mcr)
	cardEnd := cardStart + int64(mcr)
	for i := cardStart; i < cardEnd; i++ {
		if view.packedCardRefPos[i] >= 0 {
			view.packedCardRefPos[i] += delta
		}
	}

	mb := cfg.blankMaxBlanks
	blankStart := batchIdx * int64(mb)
	blankEnd := blankStart + int64(mb)
	for i := blankStart; i < blankEnd; i++ {
		if view.packedBlankPos[i] >= 0 {
			view.packedBlankPos[i] += delta
		}
	}
}

func decisionRowsForPending(pending *apiPending, cfg encodeConfig) int64 {
	traceKind := traceKindForPending(pending)
	switch traceKind {
	case "may":
		return 0
	case "priority":
		optionCount := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		count := minInt64(priorityCandidateCount(pending, optionCount, cfg.maxTargetsPerOption), cfg.maxCachedChoices)
		if count == 0 {
			return 0
		}
		return 1
	case "attackers", "blockers":
		return minInt64(int64(len(pending.Options)), cfg.maxOptions)
	default:
		optionCount := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		if optionCount == 0 {
			return 0
		}
		return 1
	}
}

func resolvePerspectivePlayerIndex(state *apiGameState, pending *apiPending, requested int64) (int, *encodeError) {
	if requested >= 0 {
		if requested >= int64(len(state.Players)) {
			return 0, &encodeError{code: mageEncodeErrPlayer, message: fmt.Sprintf("perspective_player_idx=%d outside players list", requested)}
		}
		return int(requested), nil
	}
	if pending != nil && pending.PlayerIdx >= 0 && pending.PlayerIdx < len(state.Players) {
		return pending.PlayerIdx, nil
	}
	for idx, player := range state.Players {
		if player.Name == state.ActivePlayer || player.ID.String() == state.ActivePlayer {
			return idx, nil
		}
	}
	return 0, nil
}

func fillStateEncoding(batchIdx int64, state *apiGameState, pending *apiPending, playerIdx int, cfg encodeConfig, view outputViews, cardIDToSlot map[string]int64) *encodeError {
	slotRows := view.slotCardRows[batchIdx*cfg.zoneSlotCount : (batchIdx+1)*cfg.zoneSlotCount]
	slotOccupied := view.slotOccupied[batchIdx*cfg.zoneSlotCount : (batchIdx+1)*cfg.zoneSlotCount]
	slotTapped := view.slotTapped[batchIdx*cfg.zoneSlotCount : (batchIdx+1)*cfg.zoneSlotCount]
	gameInfo := view.gameInfo[batchIdx*cfg.gameInfoDim : (batchIdx+1)*cfg.gameInfoDim]

	occupied := make([]float32, int(cfg.zoneSlotCount))
	for slotIdx, card := range collectSlotCards(state, playerIdx) {
		if card == nil {
			continue
		}
		row, ok := cardRowForName(card.name)
		if !ok {
			return &encodeError{code: mageEncodeErrEncode, message: fmt.Sprintf("missing card embedding for %q", card.name)}
		}
		slotRows[slotIdx] = row
		slotOccupied[slotIdx] = 1
		occupied[slotIdx] = 1
		if card.tapped && zoneSpecs[slotIdx/maxCardsPerZone].zone == "battlefield" {
			slotTapped[slotIdx] = 1
		}
		if card.id != "" {
			cardIDToSlot[card.id] = int64(slotIdx)
		}
	}

	fillGameInfo(gameInfo, state, pending, playerIdx, occupied)
	return nil
}

func collectSlotCards(state *apiGameState, perspectivePlayerIdx int) []*stateCard {
	player := state.Players[perspectivePlayerIdx]
	var opponent *interactive.PlayerState
	if len(state.Players) == 2 {
		opponent = &state.Players[1-perspectivePlayerIdx]
	}

	out := make([]*stateCard, 0, zoneSlotCount)
	for _, spec := range zoneSpecs {
		var cards []*stateCard
		switch spec.owner {
		case "self":
			cards = zoneCards(&player, spec.zone)
		default:
			cards = zoneCards(opponent, spec.zone)
		}
		for slotIdx := range maxCardsPerZone {
			if slotIdx < len(cards) {
				out = append(out, cards[slotIdx])
			} else {
				out = append(out, nil)
			}
		}
	}
	return out
}

func zoneCards(player *interactive.PlayerState, zone string) []*stateCard {
	if player == nil {
		return nil
	}
	switch zone {
	case "hand":
		out := make([]*stateCard, 0, minInt(len(player.Hand), maxCardsPerZone))
		for idx, card := range player.Hand {
			if idx >= maxCardsPerZone {
				break
			}

			out = append(out, &stateCard{id: card.ID.String(), name: card.Name})
		}
		return out
	case "graveyard":
		out := make([]*stateCard, 0, minInt(len(player.Graveyard), maxCardsPerZone))
		for idx, card := range player.Graveyard {
			if idx >= maxCardsPerZone {
				break
			}

			out = append(out, &stateCard{id: card.ID.String(), name: card.Name})
		}
		return out
	default:
		out := make([]*stateCard, 0, minInt(len(player.Battlefield), maxCardsPerZone))
		for idx, perm := range player.Battlefield {
			if idx >= maxCardsPerZone {
				break
			}

			out = append(out, &stateCard{id: perm.ID.String(), name: perm.Name, tapped: perm.Tapped})
		}
		return out
	}
}

func fillGameInfo(out []float32, state *apiGameState, pending *apiPending, perspectivePlayerIdx int, occupied []float32) {
	selfPlayer := state.Players[perspectivePlayerIdx]
	var opponent *interactive.PlayerState
	if len(state.Players) == 2 {
		opponent = &state.Players[1-perspectivePlayerIdx]
	}

	cursor := 0
	out[cursor] = clipNorm(float64(state.Turn), maxTurn)
	cursor++
	if state.ActivePlayer == selfPlayer.Name || state.ActivePlayer == selfPlayer.ID.String() {
		out[cursor] = 1
	}
	cursor++
	if pending != nil && pending.PlayerIdx == perspectivePlayerIdx {
		out[cursor] = 1
	}
	cursor++
	out[cursor] = clipNorm(float64(selfPlayer.Life), maxLife)
	cursor++
	if opponent != nil {
		out[cursor] = clipNorm(float64(opponent.Life), maxLife)
	}
	cursor++

	for _, player := range []*interactive.PlayerState{&selfPlayer, opponent} {
		if player == nil {
			cursor += 4
			continue
		}
		out[cursor] = clipNorm(float64(player.HandCount), maxCardsPerZone)
		out[cursor+1] = clipNorm(float64(player.GraveyardCount), maxCardsPerZone)
		out[cursor+2] = clipNorm(float64(len(player.Battlefield)), maxCardsPerZone)
		out[cursor+3] = clipNorm(float64(player.LibraryCount), maxLibrary)
		cursor += 4
	}

	for _, player := range []*interactive.PlayerState{&selfPlayer, opponent} {
		if player == nil {
			cursor += 6
			continue
		}
		out[cursor] = clipNorm(float64(player.ManaPool.White), maxMana)
		out[cursor+1] = clipNorm(float64(player.ManaPool.Blue), maxMana)
		out[cursor+2] = clipNorm(float64(player.ManaPool.Black), maxMana)
		out[cursor+3] = clipNorm(float64(player.ManaPool.Red), maxMana)
		out[cursor+4] = clipNorm(float64(player.ManaPool.Green), maxMana)
		out[cursor+5] = clipNorm(float64(player.ManaPool.Colorless), maxMana)
		cursor += 6
	}

	pendingOptionCount := 0.0
	if pending != nil {
		pendingOptionCount = float64(len(pending.Options))
	}
	out[cursor] = clipNorm(pendingOptionCount, maxPendingOpt)
	out[cursor+1] = clipNorm(float64(len(state.Stack)), maxPendingOpt)
	cursor += 2

	stepIdx := len(stepNames) - 1
	normalizedStep := normalizeKey(state.Step)
	for idx, stepKey := range stepNamesNorm[:len(stepNamesNorm)-1] {
		if stepKey == normalizedStep {
			stepIdx = idx
			break
		}
	}
	out[cursor+stepIdx] = 1
	cursor += len(stepNames)

	copy(out[cursor:], occupied)
}

func fillActionEncoding(batchIdx int64, state *apiGameState, pending *apiPending, playerIdx int, cfg encodeConfig, view outputViews, cardIDToSlot map[string]int64) *encodeError {
	view.pendingKindID[batchIdx] = indexOrUnknown(pendingKinds[:], pending.Kind)
	traceKind := traceKindForPending(pending)
	view.traceKindID[batchIdx] = indexOrUnknown(traceKinds[:], traceKind)
	if traceKind == "may" {
		view.mayMask[batchIdx] = 1
	}

	options := pending.Options
	numPresent := minInt64(int64(len(options)), cfg.maxOptions)
	view.numPresentOptions[batchIdx] = numPresent
	selfID, oppID := playerIDs(state, playerIdx)
	maxTargetScalar := maxFloat64(1, float64(cfg.maxTargetsPerOption-1))

	optionKindIDs := view.optionKindIDs[batchIdx*cfg.maxOptions : (batchIdx+1)*cfg.maxOptions]
	optionScalars := view.optionScalars[batchIdx*cfg.maxOptions*cfg.optionScalarDim : (batchIdx+1)*cfg.maxOptions*cfg.optionScalarDim]
	optionMask := view.optionMask[batchIdx*cfg.maxOptions : (batchIdx+1)*cfg.maxOptions]
	optionRefSlotIdx := view.optionRefSlotIdx[batchIdx*cfg.maxOptions : (batchIdx+1)*cfg.maxOptions]
	optionRefCardRow := view.optionRefCardRow[batchIdx*cfg.maxOptions : (batchIdx+1)*cfg.maxOptions]
	targetMask := view.targetMask[batchIdx*cfg.maxOptions*cfg.maxTargetsPerOption : (batchIdx+1)*cfg.maxOptions*cfg.maxTargetsPerOption]
	targetTypeIDs := view.targetTypeIDs[batchIdx*cfg.maxOptions*cfg.maxTargetsPerOption : (batchIdx+1)*cfg.maxOptions*cfg.maxTargetsPerOption]
	targetScalars := view.targetScalars[batchIdx*cfg.maxOptions*cfg.maxTargetsPerOption*cfg.targetScalarDim : (batchIdx+1)*cfg.maxOptions*cfg.maxTargetsPerOption*cfg.targetScalarDim]
	targetOverflow := view.targetOverflow[batchIdx*cfg.maxOptions : (batchIdx+1)*cfg.maxOptions]
	targetRefSlotIdx := view.targetRefSlotIdx[batchIdx*cfg.maxOptions*cfg.maxTargetsPerOption : (batchIdx+1)*cfg.maxOptions*cfg.maxTargetsPerOption]
	targetRefIsPlayer := view.targetRefIsPlayer[batchIdx*cfg.maxOptions*cfg.maxTargetsPerOption : (batchIdx+1)*cfg.maxOptions*cfg.maxTargetsPerOption]
	targetRefIsSelf := view.targetRefIsSelf[batchIdx*cfg.maxOptions*cfg.maxTargetsPerOption : (batchIdx+1)*cfg.maxOptions*cfg.maxTargetsPerOption]

	for optIdx := range numPresent {
		option := options[optIdx]
		optionKindIDs[optIdx] = indexOrUnknown(actionKinds[:], option.Kind)
		optionMask[optIdx] = 1
		fillOptionScalars(optionScalars[optIdx*cfg.optionScalarDim:(optIdx+1)*cfg.optionScalarDim], option, pending, optIdx, cfg)
		slotIdx, cardRow, err := resolveOptionReference(option, cardIDToSlot)
		if err != nil {
			return err
		}
		if slotIdx >= 0 {
			optionRefSlotIdx[optIdx] = slotIdx
		} else if option.CardName != "" {
			optionRefCardRow[optIdx] = cardRow
		}

		targets := option.ValidTargets
		targetOverflow[optIdx] = clipNorm(float64(maxInt(0, len(targets)-int(cfg.maxTargetsPerOption))), maxTargetOverflow)
		targetBase := optIdx * cfg.maxTargetsPerOption
		targetScalarBase := optIdx * cfg.maxTargetsPerOption * cfg.targetScalarDim
		for tgtIdx := int64(0); tgtIdx < minInt64(int64(len(targets)), cfg.maxTargetsPerOption); tgtIdx++ {
			target := targets[tgtIdx]
			targetMask[targetBase+tgtIdx] = 1
			targetScalars[targetScalarBase+tgtIdx*cfg.targetScalarDim] = clipNorm(float64(tgtIdx), maxTargetScalar)
			targetScalars[targetScalarBase+tgtIdx*cfg.targetScalarDim+1] = 1

			if target.IDUUID != uuid.Nil && (target.IDUUID == selfID || target.IDUUID == oppID) {
				targetTypeIDs[targetBase+tgtIdx] = 0
				targetRefIsPlayer[targetBase+tgtIdx] = 1
				if target.IDUUID == selfID {
					targetRefIsSelf[targetBase+tgtIdx] = 1
				}
				continue
			}
			if slot, ok := cardIDToSlot[target.ID]; ok {
				targetTypeIDs[targetBase+tgtIdx] = 1
				targetRefSlotIdx[targetBase+tgtIdx] = slot
			}
		}
	}
	return nil
}

func fillOptionScalars(out []float32, option apiOption, pending *apiPending, optionIdx int64, cfg encodeConfig) {
	targetCount := len(option.ValidTargets)
	overflowCount := maxInt(0, targetCount-int(cfg.maxTargetsPerOption))
	out[0] = clipNorm(float64(optionIdx), maxFloat64(1, float64(cfg.maxOptions-1)))
	out[1] = clipNorm(float64(option.AbilityIndex), maxAbilityIndex)
	out[2] = clipNorm(float64(targetCount), float64(cfg.maxTargetsPerOption))
	out[3] = clipNorm(float64(overflowCount), maxTargetOverflow)
	if option.CardID != "" {
		out[4] = 1
	}
	if option.PermanentID != "" {
		out[5] = 1
	}
	if option.ID != "" {
		out[6] = 1
	}
	fillManaCostFeatures(out[7:13], option.ManaCost)
	if pending != nil {
		out[13] = clipNorm(float64(pending.Amount), maxAmount)
	}
}

func fillManaCostFeatures(out []float32, manaCost string) {
	counts := map[string]float64{"W": 0, "U": 0, "B": 0, "R": 0, "G": 0, "C": 0}
	generic := 0.0
	for _, symbol := range manaSymbolsFromCost(manaCost) {
		if _, ok := counts[symbol]; ok {
			counts[symbol]++
			continue
		}
		if isDigits(symbol) {
			generic += parsePositiveFloat(symbol)
			continue
		}
		if strings.Contains(symbol, "/") {
			for part := range strings.SplitSeq(symbol, "/") {
				if _, ok := counts[part]; ok {
					counts[part] += 0.5
				}
			}
		}
	}
	counts["C"] += generic
	for idx, symbol := range manaSymbols {
		out[idx] = clipNorm(counts[symbol], 10)
	}
}

func manaSymbolsFromCost(manaCost string) []string {
	var out []string
	rest := manaCost
	for {
		start := strings.IndexByte(rest, '{')
		if start < 0 {
			return out
		}
		rest = rest[start+1:]
		end := strings.IndexByte(rest, '}')
		if end < 0 {
			return out
		}
		out = append(out, strings.ToUpper(rest[:end]))
		rest = rest[end+1:]
	}
}

func resolveOptionReference(option apiOption, cardIDToSlot map[string]int64) (int64, int64, *encodeError) {
	for _, key := range []string{option.CardID, option.PermanentID, option.ID} {
		if key == "" {
			continue
		}
		if slotIdx, ok := cardIDToSlot[key]; ok {
			return slotIdx, -1, nil
		}
	}
	if option.CardName != "" {
		row, ok := cardRowForName(option.CardName)
		if !ok {
			return -1, -1, &encodeError{code: mageEncodeErrEncode, message: fmt.Sprintf("missing card embedding for %q", option.CardName)}
		}
		return -1, row, nil
	}
	return -1, -1, nil
}

func fillDecisionEncoding(batchIdx int64, pending *apiPending, cfg encodeConfig, view outputViews, cursor int64) (int64, *encodeError) {
	traceKind := traceKindForPending(pending)
	view.decisionStart[batchIdx] = cursor

	switch traceKind {
	case "may":
		return 0, nil
	case "priority":
		optionCount := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		count := minInt64(priorityCandidateCount(pending, optionCount, cfg.maxTargetsPerOption), cfg.maxCachedChoices)
		if count == 0 {
			return 0, nil
		}
		if cursor+1 > cfg.decisionCapacity {
			return 0, &encodeError{code: mageEncodeErrBuffer, message: "decision_capacity too small for priority rows"}
		}
		rowBase := cursor * cfg.maxCachedChoices
		candidateIdx := int64(0)
		for optIdx := range optionCount {
			option := pending.Options[optIdx]
			switch option.Kind {
			case "pass", "play_land":
				if candidateIdx >= count {
					break
				}
				view.decisionOptionIdx[rowBase+candidateIdx] = optIdx
				view.decisionMask[rowBase+candidateIdx] = 1
				candidateIdx++
			case "cast_spell", "activate_ability":
				if len(option.ValidTargets) == 0 {
					if candidateIdx >= count {
						break
					}
					view.decisionOptionIdx[rowBase+candidateIdx] = optIdx
					view.decisionMask[rowBase+candidateIdx] = 1
					candidateIdx++
					continue
				}
				for tgtIdx := 0; tgtIdx < len(option.ValidTargets) && int64(tgtIdx) < cfg.maxTargetsPerOption; tgtIdx++ {
					if candidateIdx >= count {
						break
					}
					view.decisionOptionIdx[rowBase+candidateIdx] = optIdx
					view.decisionTargetIdx[rowBase+candidateIdx] = int64(tgtIdx)
					view.decisionMask[rowBase+candidateIdx] = 1
					candidateIdx++
				}
			}
			if candidateIdx >= count {
				break
			}
		}
		view.decisionCount[batchIdx] = 1
		return 1, nil
	case "attackers":
		optionCount := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		if optionCount == 0 {
			return 0, nil
		}
		if cursor+optionCount > cfg.decisionCapacity {
			return 0, &encodeError{code: mageEncodeErrBuffer, message: "decision_capacity too small for attacker rows"}
		}
		for optIdx := range optionCount {
			rowBase := (cursor + optIdx) * cfg.maxCachedChoices
			view.decisionMask[rowBase] = 1
			view.decisionMask[rowBase+1] = 1
			view.decisionOptionIdx[rowBase+1] = optIdx
			view.usesNoneHead[cursor+optIdx] = 1
		}
		view.decisionCount[batchIdx] = optionCount
		return optionCount, nil
	case "blockers":
		optionCount := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		if optionCount == 0 {
			return 0, nil
		}
		if cursor+optionCount > cfg.decisionCapacity {
			return 0, &encodeError{code: mageEncodeErrBuffer, message: "decision_capacity too small for blocker rows"}
		}
		for optIdx := range optionCount {
			rowBase := (cursor + optIdx) * cfg.maxCachedChoices
			view.decisionMask[rowBase] = 1
			view.usesNoneHead[cursor+optIdx] = 1
			targetCount := minInt64(int64(len(pending.Options[optIdx].ValidTargets)), cfg.maxTargetsPerOption)
			for tgtIdx := range targetCount {
				col := tgtIdx + 1
				view.decisionOptionIdx[rowBase+col] = optIdx
				view.decisionTargetIdx[rowBase+col] = tgtIdx
				view.decisionMask[rowBase+col] = 1
			}
		}
		view.decisionCount[batchIdx] = optionCount
		return optionCount, nil
	default:
		optionCount := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		if optionCount == 0 {
			return 0, nil
		}
		if cursor+1 > cfg.decisionCapacity {
			return 0, &encodeError{code: mageEncodeErrBuffer, message: "decision_capacity too small for choice rows"}
		}
		rowBase := cursor * cfg.maxCachedChoices
		for optIdx := range optionCount {
			view.decisionOptionIdx[rowBase+optIdx] = optIdx
			view.decisionMask[rowBase+optIdx] = 1
		}
		view.decisionCount[batchIdx] = 1
		return 1, nil
	}
}

func traceKindForPending(pending *apiPending) string {
	if pending == nil {
		return "choice_index"
	}
	switch pending.Kind {
	case "priority":
		return "priority"
	case "attackers":
		return "attackers"
	case "blockers":
		return "blockers"
	case "may":
		return "may"
	case "mana_color":
		return "choice_color"
	case "cards_from_hand", "card_from_library", "permanent":
		return "choice_ids"
	default:
		return "choice_index"
	}
}

func priorityCandidateCount(pending *apiPending, maxOptions int64, maxTargetsPerOption int64) int64 {
	if pending == nil || pending.Kind != "priority" {
		return 0
	}
	count := int64(0)
	optionCount := minInt64(int64(len(pending.Options)), maxOptions)
	for optIdx := range optionCount {
		option := pending.Options[optIdx]
		switch option.Kind {
		case "pass", "play_land":
			count++
		case "cast_spell", "activate_ability":
			if len(option.ValidTargets) == 0 {
				count++
				continue
			}
			count += minInt64(int64(len(option.ValidTargets)), maxTargetsPerOption)
		}
	}
	return count
}

func playerIDs(state *apiGameState, perspectivePlayerIdx int) (uuid.UUID, uuid.UUID) {
	var selfID, oppID uuid.UUID
	if len(state.Players) > 0 {
		selfID = state.Players[perspectivePlayerIdx].ID
	}
	if len(state.Players) == 2 {
		oppID = state.Players[1-perspectivePlayerIdx].ID
	}
	return selfID, oppID
}

func indexOrUnknown(values []string, value string) int64 {
	if cache := indexCacheFor(values); cache != nil {
		if cached, ok := cache.Load(value); ok {
			return cached.(int64)
		}
		idx := indexOrUnknownSlow(values, value)
		cache.Store(value, idx)
		return idx
	}
	return indexOrUnknownSlow(values, value)
}

func indexOrUnknownSlow(values []string, value string) int64 {
	key := normalizeKey(value)
	norm := normalizedKeysFor(values)
	for idx, candidate := range norm {
		if candidate == key {
			return int64(idx)
		}
	}
	return int64(len(values) - 1)
}

// indexCacheFor returns a per-table sync.Map memoizing prior raw-input
// lookups. Hot-path callers (the encoder dispatches an option-kind lookup
// per option, per row, per batch) avoid the strings.Fields cost that way.
// Returns nil for unknown tables so the slow path stays correct.
var (
	stepNamesIndexCache    sync.Map
	pendingKindsIndexCache sync.Map
	actionKindsIndexCache  sync.Map
	traceKindsIndexCache   sync.Map
)

func indexCacheFor(values []string) *sync.Map {
	switch {
	case len(values) == len(stepNames) && &values[0] == &stepNames[0]:
		return &stepNamesIndexCache
	case len(values) == len(pendingKinds) && &values[0] == &pendingKinds[0]:
		return &pendingKindsIndexCache
	case len(values) == len(actionKinds) && &values[0] == &actionKinds[0]:
		return &actionKindsIndexCache
	case len(values) == len(traceKinds) && &values[0] == &traceKinds[0]:
		return &traceKindsIndexCache
	}
	return nil
}

// normalizedKeys returns a slice of normalized keys, one per input value.
func normalizedKeys(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = normalizeKey(v)
	}
	return out
}

// normalizedKeysFor maps the well-known shared lookup tables to their
// precomputed normalized-key slice. Falls back to fresh normalization
// for unknown inputs (rare on the hot path).
func normalizedKeysFor(values []string) []string {
	switch {
	case len(values) == len(stepNames) && &values[0] == &stepNames[0]:
		return stepNamesNorm
	case len(values) == len(pendingKinds) && &values[0] == &pendingKinds[0]:
		return pendingKindsNorm
	case len(values) == len(actionKinds) && &values[0] == &actionKinds[0]:
		return actionKindsNorm
	case len(values) == len(traceKinds) && &values[0] == &traceKinds[0]:
		return traceKindsNorm
	}
	return normalizedKeys(values)
}

func normalizeKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func cardRowForName(name string) (int64, bool) {
	if name == "" {
		return 0, true
	}
	// Stack entries for activated/triggered abilities can arrive without an
	// inspectable source card name. They are still legitimate visible stack
	// objects, so encode them with the row-0 unknown-card sentinel instead of
	// looking them up in the real-card embedding table.
	if normalizeKey(name) == "ability" {
		return 0, true
	}

	cardRowOverrideMu.RLock()
	overridden := cardRowsOverridden
	if overridden {
		if row, ok := cardRowOverridesByRaw[name]; ok {
			cardRowOverrideMu.RUnlock()
			return row, true
		}
		key := normalizeKey(name)
		row, ok := cardRowOverrides[key]
		cardRowOverrideMu.RUnlock()
		return row, ok
	}
	cardRowOverrideMu.RUnlock()

	cardRowsOnce.Do(func() {
		names := mage.RegisteredCardNames()
		sort.Strings(names)
		cardRowByName = make(map[string]int64, len(names))
		cardRowByRawName = make(map[string]int64, len(names))
		for idx, cardName := range names {
			cardRowByName[normalizeKey(cardName)] = int64(idx + 1)
			cardRowByRawName[cardName] = int64(idx + 1)
		}
	})
	if row, ok := cardRowByRawName[name]; ok {
		return row, true
	}
	row, ok := cardRowByName[normalizeKey(name)]
	if !ok {
		return 0, true
	}
	return row, true
}

func setCardRowOverrides(rows map[string]int64) {
	next := make(map[string]int64, len(rows))
	nextRaw := make(map[string]int64, len(rows))
	for name, row := range rows {
		next[normalizeKey(name)] = row
		nextRaw[name] = row
	}
	cardRowOverrideMu.Lock()
	cardRowOverrides = next
	cardRowOverridesByRaw = nextRaw
	cardRowsOverridden = true
	cardRowOverrideMu.Unlock()
}

func fillInt64(dst []int64, value int64) {
	for i := range dst {
		dst[i] = value
	}
}

func fillFloat32(dst []float32, value float32) {
	for i := range dst {
		dst[i] = value
	}
}

func fillBytes(dst []byte, value byte) {
	for i := range dst {
		dst[i] = value
	}
}

func fillInt32(dst []int32, value int32) {
	for i := range dst {
		dst[i] = value
	}
}

func clipNorm(value float64, maximum float64) float32 {
	if maximum <= 0 {
		return 0
	}
	if value < 0 {
		value = 0
	}
	if value > maximum {
		value = maximum
	}
	return float32(value / maximum)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parsePositiveFloat(s string) float64 {
	var value float64
	for _, r := range s {
		value = value*10 + float64(r-'0')
	}
	return value
}
