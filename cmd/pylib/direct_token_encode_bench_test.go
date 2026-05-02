package main

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// Benchmark for the direct token-encode path (the production path that
// magic-ai's encode_tokens_packed actually exercises: emit_render_plan=0
// in native_assembler.py forces directTokenEncode mode).
//
// Mirrors the per-row work in fillTokenAssemblyDirectPacked: build the
// renderPlanIndex from a per-batch apiGameState/apiPending, then run
// directTokenEmitter through emitDirectTokens, into a packed B*max_tokens
// buffer.
//
// Row-shape parameters match BenchmarkAssembleTokensPackedBatch so
// numbers are directly comparable.

const (
	benchDirectBattlefieldCount = 12
	benchDirectHandCount        = 4
	benchDirectGraveyardCount   = 2
)

func benchDirectGameState(seed int) *apiGameState {
	mkPerm := func(rowIdx int, ownerIdx int, slot int) interactive.PermanentState {
		name := fmt.Sprintf("BenchCard%d", rowIdx%benchCardRowCount)
		return interactive.PermanentState{
			ID:         uuid.New(),
			Name:       name,
			Power:      2,
			Toughness:  2,
			Tapped:     (slot+ownerIdx)%3 == 0,
			IsCreature: true,
		}
	}
	mkCard := func(rowIdx int) interactive.CardState {
		return interactive.CardState{
			ID:   uuid.New(),
			Name: fmt.Sprintf("BenchCard%d", rowIdx%benchCardRowCount),
		}
	}

	mkPlayer := func(idx int) interactive.PlayerState {
		bf := make([]interactive.PermanentState, benchDirectBattlefieldCount)
		for i := range bf {
			bf[i] = mkPerm(seed+idx*5+i, idx, i)
		}
		hand := make([]interactive.CardState, benchDirectHandCount)
		for i := range hand {
			hand[i] = mkCard(seed*3 + idx + i)
		}
		gy := make([]interactive.CardState, benchDirectGraveyardCount)
		for i := range gy {
			gy[i] = mkCard(seed*7 + idx*2 + i)
		}
		return interactive.PlayerState{
			ID:           uuid.New(),
			Name:         fmt.Sprintf("P%d", idx),
			Life:         20 - (seed % 5),
			HandCount:    len(hand),
			Hand:         hand,
			Battlefield:  bf,
			Graveyard:    gy,
			LibraryCount: 50,
			ManaPool: interactive.ManaPoolState{
				White: 1, Blue: 1, Black: 1, Red: 2, Green: 2, Colorless: 1,
			},
		}
	}

	return &apiGameState{
		Turn:         1 + (seed % 50),
		Step:         "Precombat Main",
		ActivePlayer: "P0",
		Players:      [2]interactive.PlayerState{mkPlayer(0), mkPlayer(1)},
	}
}

func benchDirectPending(state *apiGameState) *apiPending {
	selfPlayer := state.Players[0]
	oppPlayer := state.Players[1]
	options := make([]apiOption, 0, benchOptionCount)
	kinds := []string{"play_land", "cast_spell", "activate_ability", "attacker", "blocker"}
	for o := 0; o < benchOptionCount; o++ {
		k := kinds[o%len(kinds)]
		var cardID, cardName, permID string
		var cardUUID, permUUID uuid.UUID
		// alternate sourcing options between hand cards and battlefield perms
		if o%2 == 0 && len(selfPlayer.Hand) > 0 {
			c := selfPlayer.Hand[o%len(selfPlayer.Hand)]
			cardID, cardName, cardUUID = c.ID.String(), c.Name, c.ID
		} else if len(selfPlayer.Battlefield) > 0 {
			p := selfPlayer.Battlefield[o%len(selfPlayer.Battlefield)]
			permID, cardName, permUUID = p.ID.String(), p.Name, p.ID
		}
		opt := apiOption{
			Kind:          k,
			CardID:        cardID,
			CardName:      cardName,
			PermanentID:   permID,
			AbilityIndex:  o % 8,
			CardUUID:      cardUUID,
			PermanentUUID: permUUID,
		}
		// 2 targets each — first a player target, second a permanent
		permTgt := oppPlayer.Battlefield[o%len(oppPlayer.Battlefield)]
		opt.ValidTargets = []apiTarget{
			{ID: oppPlayer.ID.String(), Label: "opp", IDUUID: oppPlayer.ID},
			{ID: permTgt.ID.String(), Label: "perm", IDUUID: permTgt.ID},
		}
		options = append(options, opt)
	}
	return &apiPending{Kind: "priority", PlayerIdx: 0, Options: options}
}

// benchRegisterCardRows seeds the global card-row override map with the
// names produced by benchDirectGameState. Required because the direct
// emitter looks up rows by name through cardRowForName.
func benchRegisterCardRows() {
	rows := make(map[string]int64, benchCardRowCount)
	for i := 0; i < benchCardRowCount; i++ {
		rows[fmt.Sprintf("BenchCard%d", i)] = int64(i)
	}
	setCardRowOverrides(rows)
}

func benchDirectAllocOutputs() outputViews {
	tokenIDs := make([]int32, benchBatchSize*benchMaxTokens)
	cuSeqlens := make([]int32, benchBatchSize+1)
	seqLengths := make([]int32, benchBatchSize)
	statePos := make([]int32, benchBatchSize)
	optionPos := make([]int32, benchBatchSize*benchMaxOptions)
	optionMask := make([]byte, benchBatchSize*benchMaxOptions)
	targetPos := make([]int32, benchBatchSize*benchMaxOptions*benchMaxTargets)
	targetMask := make([]byte, benchBatchSize*benchMaxOptions*benchMaxTargets)
	cardRefPos := make([]int32, benchBatchSize*benchMaxCardRefs)
	tokenOverflow := make([]int32, benchBatchSize)
	return outputViews{
		packedTokenIDs:       tokenIDs,
		packedCuSeqlens:      cuSeqlens,
		packedSeqLengths:     seqLengths,
		packedStatePositions: statePos,
		packedOptionPos:      optionPos,
		packedOptionMask:     optionMask,
		packedTargetPos:      targetPos,
		packedTargetMask:     targetMask,
		packedCardRefPos:     cardRefPos,
		packedTokenOverflow:  tokenOverflow,
	}
}

func BenchmarkDirectTokenEncodePackedBatch(b *testing.B) {
	tables := benchTokenTables()
	tokenTablesMu.Lock()
	prev := currentTokenTables
	currentTokenTables = tables
	tokenTablesMu.Unlock()
	defer func() {
		tokenTablesMu.Lock()
		currentTokenTables = prev
		tokenTablesMu.Unlock()
	}()

	benchRegisterCardRows()

	states := make([]*apiGameState, benchBatchSize)
	pendings := make([]*apiPending, benchBatchSize)
	for i := 0; i < benchBatchSize; i++ {
		states[i] = benchDirectGameState(i)
		pendings[i] = benchDirectPending(states[i])
	}

	view := benchDirectAllocOutputs()

	cfg := encodeConfig{
		maxOptions:          benchMaxOptions,
		maxTargetsPerOption: benchMaxTargets,
		tokenMaxTokens:      benchMaxTokens,
		tokenMaxOptions:     benchMaxOptions,
		tokenMaxTargets:     benchMaxTargets,
		tokenMaxCardRefs:    benchMaxCardRefs,
		dedupCardBodies:     true,
	}

	// Mirror the production path: acquire a scratch from the per-output
	// pool at the start of each call and release it at the end. With a
	// stable output buffer, we always get the same scratch back, so its
	// directDirty state survives across calls and the high-water-mark
	// partial clears in directTokenEmitter.reset stay warm.
	poolKey := scratchPoolKey(view)

	// Warm up + sanity-check one row so the bench fails fast on plumbing
	// errors before timing.
	{
		warm := acquireScratch(poolKey)
		warm.reset()
		_, _, err := fillTokenAssemblyDirectPacked(0, 0, states[0], pendings[0], 0, cfg, view, warm)
		if err != nil {
			b.Fatalf("warmup direct encode: %s", err.message)
		}
		releaseScratch(poolKey, warm)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		var packedCursor int32
		view.packedCuSeqlens[0] = 0
		scratch := acquireScratch(poolKey)
		for i := 0; i < benchBatchSize; i++ {
			scratch.reset()
			next, _, err := fillTokenAssemblyDirectPacked(int64(i), packedCursor, states[i], pendings[i], 0, cfg, view, scratch)
			if err != nil {
				b.Fatalf("row=%d direct encode: %s", i, err.message)
			}
			packedCursor = next
		}
		releaseScratch(poolKey, scratch)
		if iter == 0 {
			b.Logf("packed cursor after batch = %d / %d", packedCursor, benchBatchSize*benchMaxTokens)
		}
	}
}

// BenchmarkDirectTokenEncodePackedBatchNoPool reproduces the old
// scratch-per-call behavior so the pool's effect is measurable: every
// iteration allocates a fresh encodeScratch, which forces every row's
// directDirty entry to start "uninitialized" -> full per-row clear in
// directTokenEmitter.reset.
func BenchmarkDirectTokenEncodePackedBatchNoPool(b *testing.B) {
	tables := benchTokenTables()
	tokenTablesMu.Lock()
	prev := currentTokenTables
	currentTokenTables = tables
	tokenTablesMu.Unlock()
	defer func() {
		tokenTablesMu.Lock()
		currentTokenTables = prev
		tokenTablesMu.Unlock()
	}()

	benchRegisterCardRows()

	states := make([]*apiGameState, benchBatchSize)
	pendings := make([]*apiPending, benchBatchSize)
	for i := 0; i < benchBatchSize; i++ {
		states[i] = benchDirectGameState(i)
		pendings[i] = benchDirectPending(states[i])
	}

	view := benchDirectAllocOutputs()

	cfg := encodeConfig{
		maxOptions:          benchMaxOptions,
		maxTargetsPerOption: benchMaxTargets,
		tokenMaxTokens:      benchMaxTokens,
		tokenMaxOptions:     benchMaxOptions,
		tokenMaxTargets:     benchMaxTargets,
		tokenMaxCardRefs:    benchMaxCardRefs,
		dedupCardBodies:     true,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		var packedCursor int32
		view.packedCuSeqlens[0] = 0
		scratch := newEncodeScratch()
		for i := 0; i < benchBatchSize; i++ {
			scratch.reset()
			next, _, err := fillTokenAssemblyDirectPacked(int64(i), packedCursor, states[i], pendings[i], 0, cfg, view, scratch)
			if err != nil {
				b.Fatalf("row=%d direct encode: %s", i, err.message)
			}
			packedCursor = next
		}
	}
}
