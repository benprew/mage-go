package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

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
)

var (
	manaSymbols        = [...]string{"W", "U", "B", "R", "G", "C"}
	stepNames          = [...]string{"Untap", "Upkeep", "Draw", "Precombat Main", "Begin Combat", "Declare Attackers", "Declare Blockers", "Combat Damage", "End Combat", "Postcombat Main", "End", "Cleanup", "Unknown"}
	pendingKinds       = [...]string{"priority", "attackers", "blockers", "permanent", "cards_from_hand", "mana_color", "card_from_library", "may", "mode", "number", "unknown"}
	actionKinds        = [...]string{"pass", "play_land", "cast_spell", "activate_ability", "attacker", "blocker", "choice", "unknown"}
	traceKinds         = [...]string{"priority", "attackers", "blockers", "choice_index", "choice_ids", "choice_color", "may"}
	zoneSpecs          = [...]zoneSpec{{zone: "hand", owner: "self"}, {zone: "graveyard", owner: "self"}, {zone: "graveyard", owner: "opponent"}, {zone: "battlefield", owner: "self"}, {zone: "battlefield", owner: "opponent"}}
	stepNamesNorm      = normalizedKeys(stepNames[:])
	pendingKindsNorm   = normalizedKeys(pendingKinds[:])
	actionKindsNorm    = normalizedKeys(actionKinds[:])
	traceKindsNorm     = normalizedKeys(traceKinds[:])
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
	dedupCardBodies bool
	// emitTokens turns on the native token-assembler pass after the
	// render-plan emission. Output buffers live in tokenAssemblerViews.
	emitTokens       bool
	tokenMaxTokens   int32
	tokenMaxOptions  int32
	tokenMaxTargets  int32
	tokenMaxCardRefs int32
	// emitTokensPacked is the varlen sibling of ``emitTokens``. Only one
	// of the two flags may be set per encode call. When set, the packed
	// output buffers in ``outputViews`` are filled instead.
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

	// Token-assembler outputs. nil when emit_tokens=false.
	tokenIDs        []int64
	tokenAttention  []int64
	tokenSeqLengths []int64
	tokenOptionPos  []int64
	tokenOptionMask []byte
	tokenTargetPos  []int64
	tokenTargetMask []byte
	tokenCardRefPos []int64
	tokenOverflow   []int32

	// Packed (varlen) token-assembler outputs. Mutually exclusive with
	// the dense ``token*`` views above: only one of the two paths is
	// active per encode call. ``packedTokenIDs`` etc. are sized
	// [B*max_tokens]; ``packedSeqId`` and ``packedPosInSeq`` likewise.
	packedTokenIDs        []int64
	packedSeqID           []int64
	packedPosInSeq        []int64
	packedCuSeqlens       []int64 // [B+1]
	packedSeqLengths      []int64 // [B]
	packedStatePositions  []int64 // [B]
	packedOptionPos       []int64
	packedOptionMask      []byte
	packedTargetPos       []int64
	packedTargetMask      []byte
	packedCardRefPos      []int64
	packedTokenOverflow   []int32
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
	case cfg.emitRenderPlan && cfg.renderPlanCapacity <= 0:
		return &encodeError{code: mageEncodeErrArg, message: "render_plan_capacity must be positive when emit_render_plan is set"}
	case cfg.renderPlanCapacity > math.MaxInt32:
		return &encodeError{code: mageEncodeErrArg, message: "render_plan_capacity must fit in int32"}
	}
	return nil
}

func encodeBatchGo(req batchRequest, cfg encodeConfig, views outputViews) (int64, *encodeError) {
	clearOutputViews(views, cfg)
	decisionCursor := int64(0)
	// Running write cursor into the packed token buffer. Only advanced
	// when emitTokensPacked is set; ignored otherwise.
	packedCursor := int32(0)
	if cfg.emitTokensPacked && len(views.packedCuSeqlens) > 0 {
		views.packedCuSeqlens[0] = 0
	}
	scratch := newEncodeScratch()
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
		if err := fillStateEncoding(int64(batchIdx), state, pending, playerIdx, cfg, views, cardIDToSlot); err != nil {
			h.mu.Unlock()
			return decisionCursor, err
		}
		if err := fillActionEncoding(int64(batchIdx), state, pending, playerIdx, cfg, views, cardIDToSlot); err != nil {
			h.mu.Unlock()
			return decisionCursor, err
		}
		if cfg.emitRenderPlan {
			if err := fillRenderPlan(int64(batchIdx), state, pending, playerIdx, cfg, views, scratch); err != nil {
				h.mu.Unlock()
				return decisionCursor, err
			}
		}
		if cfg.emitTokens {
			if err := fillTokenAssembly(int64(batchIdx), cfg, views); err != nil {
				h.mu.Unlock()
				return decisionCursor, err
			}
		}
		if cfg.emitTokensPacked {
			advanced, err := fillTokenAssemblyPacked(int64(batchIdx), packedCursor, cfg, views)
			if err != nil {
				h.mu.Unlock()
				return decisionCursor, err
			}
			packedCursor = advanced
		}
		written, err := fillDecisionEncoding(int64(batchIdx), pending, cfg, views, decisionCursor)
		h.mu.Unlock()
		if err != nil {
			return decisionCursor, err
		}
		decisionCursor += written
	}
	return decisionCursor, nil
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
	// Dense token-assembler buffers are only live when emitTokens is set;
	// zeroing them in packed mode is wasted work. The dense and packed
	// paths are mutually exclusive (validated at the C entry points), so
	// in packed mode the dense slices are typically nil anyway — skip the
	// loops outright.
	if cfg.emitTokens {
		fillInt64(view.tokenIDs, 0)
		fillInt64(view.tokenAttention, 0)
		fillInt64(view.tokenSeqLengths, 0)
		fillInt64(view.tokenOptionPos, -1)
		fillBytes(view.tokenOptionMask, 0)
		fillInt64(view.tokenTargetPos, -1)
		fillBytes(view.tokenTargetMask, 0)
		fillInt64(view.tokenCardRefPos, -1)
		fillInt32(view.tokenOverflow, 0)
	}
	if cfg.emitTokensPacked {
		// Packed buffers: clear sentinel/anchor regions. The token /
		// seq_id / pos_in_seq buffers are written contiguously up to
		// cu_seqlens[B]; their tail is unspecified, so no need to zero
		// them.
		fillInt64(view.packedCuSeqlens, 0)
		fillInt64(view.packedSeqLengths, 0)
		fillInt64(view.packedStatePositions, 0)
		fillInt64(view.packedOptionPos, -1)
		fillBytes(view.packedOptionMask, 0)
		fillInt64(view.packedTargetPos, -1)
		fillBytes(view.packedTargetMask, 0)
		fillInt64(view.packedCardRefPos, -1)
		fillInt32(view.packedTokenOverflow, 0)
	}
}

// fillTokenAssembly walks the render-plan stream emitted for “batchIdx“
// and fills the token-assembler outputs for that row. Requires that the
// render plan was already emitted (cfg.emitRenderPlan must be true).
func fillTokenAssembly(batchIdx int64, cfg encodeConfig, view outputViews) *encodeError {
	tables := getTokenTables()
	if tables == nil {
		return &encodeError{
			code:    mageEncodeErrEncodeFailure,
			message: "MageRegisterTokenTables must be called before MageEncodeTokens",
		}
	}
	planStart := batchIdx * cfg.renderPlanCapacity
	planLen := view.renderPlanLengths[batchIdx]
	plan := view.renderPlan[planStart : planStart+planLen]

	mt := int64(cfg.tokenMaxTokens)
	mo := int64(cfg.tokenMaxOptions)
	mtg := int64(cfg.tokenMaxTargets)
	mcr := int64(cfg.tokenMaxCardRefs)

	out := &tokenAssemblerOut{
		tokenIDs:      view.tokenIDs[batchIdx*mt : (batchIdx+1)*mt],
		attentionMask: view.tokenAttention[batchIdx*mt : (batchIdx+1)*mt],
		optionPos:     view.tokenOptionPos[batchIdx*mo : (batchIdx+1)*mo],
		optionMask:    view.tokenOptionMask[batchIdx*mo : (batchIdx+1)*mo],
		targetPos:     view.tokenTargetPos[batchIdx*mo*mtg : (batchIdx+1)*mo*mtg],
		targetMask:    view.tokenTargetMask[batchIdx*mo*mtg : (batchIdx+1)*mo*mtg],
		cardRefPos:    view.tokenCardRefPos[batchIdx*mcr : (batchIdx+1)*mcr],
		maxOptions:    cfg.tokenMaxOptions,
		maxTargets:    cfg.tokenMaxTargets,
		maxCardRefs:   cfg.tokenMaxCardRefs,
		cursorBase:    0,
		padTail:       true,
	}

	cursor, overflow, err := assembleTokensFromPlan(plan, tables, out, cfg.tokenMaxTokens)
	if err != nil {
		return &encodeError{code: mageEncodeErrEncodeFailure, message: err.Error()}
	}
	view.tokenSeqLengths[batchIdx] = int64(cursor)
	if overflow {
		view.tokenOverflow[batchIdx] = 1
	}
	return nil
}

// fillTokenAssemblyPacked writes one row's worth of tokens into the
// shared packed output buffer starting at ``packedCursor``. Returns the
// new cursor (one past the last live token) so the caller can chain
// rows without an outer-loop allocation. Anchors are written as
// absolute offsets into the packed buffer.
func fillTokenAssemblyPacked(
	batchIdx int64,
	packedCursor int32,
	cfg encodeConfig,
	view outputViews,
) (int32, *encodeError) {
	tables := getTokenTables()
	if tables == nil {
		return packedCursor, &encodeError{
			code:    mageEncodeErrEncodeFailure,
			message: "MageRegisterTokenTables must be called before MageEncodeTokensPacked",
		}
	}
	planStart := batchIdx * cfg.renderPlanCapacity
	planLen := view.renderPlanLengths[batchIdx]
	plan := view.renderPlan[planStart : planStart+planLen]

	mt := int64(cfg.tokenMaxTokens)
	mo := int64(cfg.tokenMaxOptions)
	mtg := int64(cfg.tokenMaxTargets)
	mcr := int64(cfg.tokenMaxCardRefs)

	// Carve a row-sized scratch slice straight out of the packed buffer
	// at the running cursor. The assembler writes tokens into this view
	// using its own 0-based local cursor; with cursorBase=packedCursor
	// the anchor positions land as absolute offsets.
	rowStart := int64(packedCursor)
	rowEnd := rowStart + mt
	if rowEnd > int64(len(view.packedTokenIDs)) {
		return packedCursor, &encodeError{
			code:    mageEncodeErrInvalidArgument,
			message: "packed token buffer too small (need >= B*max_tokens)",
		}
	}

	out := &tokenAssemblerOut{
		tokenIDs:      view.packedTokenIDs[rowStart:rowEnd],
		attentionMask: nil, // packed mode does not use attention_mask
		optionPos:     view.packedOptionPos[batchIdx*mo : (batchIdx+1)*mo],
		optionMask:    view.packedOptionMask[batchIdx*mo : (batchIdx+1)*mo],
		targetPos:     view.packedTargetPos[batchIdx*mo*mtg : (batchIdx+1)*mo*mtg],
		targetMask:    view.packedTargetMask[batchIdx*mo*mtg : (batchIdx+1)*mo*mtg],
		cardRefPos:    view.packedCardRefPos[batchIdx*mcr : (batchIdx+1)*mcr],
		maxOptions:    cfg.tokenMaxOptions,
		maxTargets:    cfg.tokenMaxTargets,
		maxCardRefs:   cfg.tokenMaxCardRefs,
		cursorBase:    packedCursor,
		padTail:       false,
	}

	cursor, overflow, err := assembleTokensFromPlan(plan, tables, out, cfg.tokenMaxTokens)
	if err != nil {
		return packedCursor, &encodeError{
			code:    mageEncodeErrEncodeFailure,
			message: err.Error(),
		}
	}

	// Per-token metadata for the live region of this row.
	for k := int32(0); k < cursor; k++ {
		view.packedSeqID[packedCursor+k] = batchIdx
		view.packedPosInSeq[packedCursor+k] = int64(k)
	}
	view.packedSeqLengths[batchIdx] = int64(cursor)
	view.packedStatePositions[batchIdx] = int64(packedCursor)
	view.packedCuSeqlens[batchIdx+1] = int64(packedCursor + cursor)
	if overflow {
		view.packedTokenOverflow[batchIdx] = 1
	}
	return packedCursor + cursor, nil
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
		for slotIdx := 0; slotIdx < maxCardsPerZone; slotIdx++ {
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
			card := card
			out = append(out, &stateCard{id: card.ID.String(), name: card.Name})
		}
		return out
	case "graveyard":
		out := make([]*stateCard, 0, minInt(len(player.Graveyard), maxCardsPerZone))
		for idx, card := range player.Graveyard {
			if idx >= maxCardsPerZone {
				break
			}
			card := card
			out = append(out, &stateCard{id: card.ID.String(), name: card.Name})
		}
		return out
	default:
		out := make([]*stateCard, 0, minInt(len(player.Battlefield), maxCardsPerZone))
		for idx, perm := range player.Battlefield {
			if idx >= maxCardsPerZone {
				break
			}
			perm := perm
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

	for optIdx := int64(0); optIdx < numPresent; optIdx++ {
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

			if target.ID != "" && (target.ID == selfID || target.ID == oppID) {
				targetTypeIDs[targetBase+tgtIdx] = 0
				targetRefIsPlayer[targetBase+tgtIdx] = 1
				if target.ID == selfID {
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
			for _, part := range strings.Split(symbol, "/") {
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
		for optIdx := int64(0); optIdx < optionCount; optIdx++ {
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
		for optIdx := int64(0); optIdx < optionCount; optIdx++ {
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
		for optIdx := int64(0); optIdx < optionCount; optIdx++ {
			rowBase := (cursor + optIdx) * cfg.maxCachedChoices
			view.decisionMask[rowBase] = 1
			view.usesNoneHead[cursor+optIdx] = 1
			targetCount := minInt64(int64(len(pending.Options[optIdx].ValidTargets)), cfg.maxTargetsPerOption)
			for tgtIdx := int64(0); tgtIdx < targetCount; tgtIdx++ {
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
		for optIdx := int64(0); optIdx < optionCount; optIdx++ {
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
	for optIdx := int64(0); optIdx < optionCount; optIdx++ {
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

func playerIDs(state *apiGameState, perspectivePlayerIdx int) (string, string) {
	selfID := ""
	if len(state.Players) > 0 {
		selfID = state.Players[perspectivePlayerIdx].ID.String()
	}
	oppID := ""
	if len(state.Players) == 2 {
		oppID = state.Players[1-perspectivePlayerIdx].ID.String()
	}
	return selfID, oppID
}

func indexOrUnknown(values []string, value string) int64 {
	key := normalizeKey(value)
	norm := normalizedKeysFor(values)
	for idx, candidate := range norm {
		if candidate == key {
			return int64(idx)
		}
	}
	return int64(len(values) - 1)
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
