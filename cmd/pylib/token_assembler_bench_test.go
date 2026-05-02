package main

import (
	"testing"
)

// Benchmark for assembleTokensFromPlan driven through the same per-row
// packed-output pattern that MageEncodeTokensPacked / fillTokenAssemblyPacked
// uses on every native call from Python (text encoder rollouts and training).
//
// Mirrors the production call shape:
//   * pre-allocated B*max_tokens token buffer
//   * per-row anchor slabs (option/target/card_ref) sized at max_options /
//     max_options*max_targets / max_card_refs
//   * cursor-based row stitching so anchors land as absolute offsets
//   * v2 plan shape (open_state, open_dict, dict_entry*, close_dict,
//     turn, open_player×2 with life/mana/zones populated by placeCardRef,
//     open_actions, option×K with target×T, close_actions, close_state)
//
// Token-table dimensions match the Python defaults in
// magic_ai/text_encoder/token_tables.py (TURN_MAX=200, LIFE range = -30..300,
// COUNT_MAX=200, ABILITY_MAX=16, MAX_CARD_REFS=256).

const (
	benchBatchSize    = 64
	benchMaxTokens    = 4096
	benchMaxOptions   = 64
	benchMaxTargets   = 4
	benchMaxCardRefs  = 256
	benchCardRowCount = 32
	benchDictEntries  = 32
	benchZoneCards    = 12
	benchOptionCount  = 24
	benchTargetCount  = 2
)

func benchTokenTables() *tokenTables {
	t := &tokenTables{
		turnMin:         0,
		turnMax:         200,
		stepCount:       12,
		lifeMin:         -30,
		lifeMax:         300,
		ownerCount:      2,
		abilityMin:      0,
		abilityMax:      16,
		countMin:        0,
		countMax:        200,
		zoneCount:       7,
		actionVerbCount: 10,
		manaColorCount:  6,
		cardRefCount:    benchMaxCardRefs,
		padID:           0,
		optionID:        9001,
		targetOpenID:    9002,
		targetCloseID:   9003,
		tappedID:        9004,
		untappedID:      9005,
		cardRowCount:    benchCardRowCount,
		dictOpenID:      9100,
		dictCloseID:     9101,
		cardOpenID:      9102,
		selfID:          9200,
		oppID:           9201,
		stackOpenID:     9300,
		stackCloseID:    9301,
		commandOpenID:   9302,
		commandCloseID:  9303,
	}

	// pack32 builds a (tokens, offsets) pair where every entry has spanLen
	// tokens drawn from a deterministic, distinct id range so that span
	// lookups and copies produce realistic per-call work.
	pack32 := func(entries int, spanLen int, baseID int32) (tokens []int32, offsets []int32) {
		offsets = make([]int32, entries+1)
		tokens = make([]int32, entries*spanLen)
		for i := 0; i < entries; i++ {
			offsets[i] = int32(i * spanLen)
			for j := 0; j < spanLen; j++ {
				tokens[i*spanLen+j] = baseID + int32(i*spanLen+j)
			}
		}
		offsets[entries] = int32(entries * spanLen)
		return
	}
	pack64 := func(entries int, spanLen int, baseID int32) (tokens []int32, offsets []int64) {
		offsets = make([]int64, entries+1)
		tokens = make([]int32, entries*spanLen)
		for i := 0; i < entries; i++ {
			offsets[i] = int64(i * spanLen)
			for j := 0; j < spanLen; j++ {
				tokens[i*spanLen+j] = baseID + int32(i*spanLen+j)
			}
		}
		offsets[entries] = int64(entries * spanLen)
		return
	}

	t.structuralTokens, t.structuralOffsets = pack32(12, 2, 10000)
	turnEntries := int((t.turnMax-t.turnMin+1)*t.stepCount)
	t.turnStepTokens, t.turnStepOff = pack32(turnEntries, 4, 20000)
	lifeEntries := int((t.lifeMax - t.lifeMin + 1) * t.ownerCount)
	t.lifeOwnerTokens, t.lifeOwnerOff = pack32(lifeEntries, 4, 30000)
	abEntries := int(t.abilityMax - t.abilityMin + 1)
	t.abilityTokens, t.abilityOff = pack32(abEntries, 2, 40000)
	cnEntries := int(t.countMax - t.countMin + 1)
	t.countTokens, t.countOff = pack32(cnEntries, 2, 50000)
	zoneEntries := int(t.zoneCount * t.ownerCount)
	t.zoneOpenTokens, t.zoneOpenOff = pack32(zoneEntries, 2, 60000)
	t.zoneCloseTok, t.zoneCloseOff = pack32(zoneEntries, 2, 61000)
	t.actionVerbTokens, t.actionVerbOff = pack32(int(t.actionVerbCount), 2, 70000)
	t.manaTokens, t.manaOff = pack32(int(t.manaColorCount), 1, 80000)

	t.cardBodyToks, t.cardBodyOff = pack64(int(t.cardRowCount), 24, 90000)
	t.cardNameToks, t.cardNameOff = pack64(int(t.cardRowCount), 4, 95000)

	t.cardRefIDs = make([]int32, t.cardRefCount)
	for i := range t.cardRefIDs {
		t.cardRefIDs[i] = 100000 + int32(i)
	}
	t.cardCloser = []int32{9400, 9401}
	t.statusTapped = []int32{9500}
	t.statusUntapped = []int32{9501}
	t.dictEntryIDs = make([]int32, t.cardRowCount)
	for i := range t.dictEntryIDs {
		t.dictEntryIDs[i] = 110000 + int32(i)
	}
	return t
}

// benchPlan builds one row's render-plan in the v2 shape produced by the
// Go-side fillRenderPlan (dict prefix, per-player zones via placeCardRef,
// actions block with options/targets).
func benchPlan(rowSeed int) []int32 {
	plan := make([]int32, 0, 1024)
	add := func(vs ...int32) { plan = append(plan, vs...) }

	add(opOpenState)
	add(opOpenDict)
	for r := int32(0); r < benchDictEntries; r++ {
		add(opDictEntry, r%benchCardRowCount)
	}
	add(opCloseDict)

	turn := int32(1 + (rowSeed % 50))
	add(opTurn, turn, 5)

	for owner := int32(0); owner < 2; owner++ {
		add(opOpenPlayer, owner)
		add(opLife, owner, int32(20-(rowSeed%5)))
		// mana: a few colors
		for c := int32(0); c < 5; c++ {
			amount := int32(1 + (rowSeed+int(c))%3)
			add(opMana, owner, c, amount)
		}
		// battlefield zone with placeCardRef entries
		add(opOpenZone, renderZoneBattlefield, owner)
		for k := 0; k < benchZoneCards; k++ {
			row := int32((rowSeed + k) % benchCardRowCount)
			status := int32(0)
			if k%3 == 0 {
				status = statusTappedKnown | 0x1
			}
			uuidIdx := int32((rowSeed*7 + k) % benchMaxCardRefs)
			add(opPlaceCardRef, int32(k), row, status, uuidIdx)
		}
		add(opCloseZone)
		// hand zone: a couple of refs
		add(opOpenZone, renderZoneHand, owner)
		for k := 0; k < 4; k++ {
			row := int32((rowSeed*3 + k) % benchCardRowCount)
			uuidIdx := int32((rowSeed*11 + k) % benchMaxCardRefs)
			add(opPlaceCardRef, int32(k), row, 0, uuidIdx)
		}
		add(opCloseZone)
		add(opClosePlayer)
	}

	add(opStackOpen)
	add(opStackClose)

	add(opOpenActions)
	for o := 0; o < benchOptionCount; o++ {
		kindID := int32(1 + (o % 5))   // skip 0 (pass) / 6 (choice) so source/ability work
		sourceRow := int32(o % benchCardRowCount)
		sourceUUID := int32((o * 13) % benchMaxCardRefs)
		manaCostID := int32(0)
		abilityIdx := int32(-1)
		if kindID == 3 {
			abilityIdx = int32(o % 16)
		}
		add(opOption, kindID, sourceRow, sourceUUID, manaCostID, abilityIdx)
		for tgt := 0; tgt < benchTargetCount; tgt++ {
			targetRow := int32((o + tgt) % benchCardRowCount)
			targetUUID := int32((o*7 + tgt) % benchMaxCardRefs)
			targetKind := int32(1) // permanent
			if tgt == 0 {
				targetKind = 0 // player
				targetRow = int32(tgt % 2)
			}
			add(opTarget, targetRow, targetUUID, targetKind)
		}
	}
	add(opCloseActions)
	add(opCloseState)
	return plan
}

func benchAllocOutputs() (
	tokenIDs []int32,
	cuSeqlens []int32,
	seqLengths []int32,
	statePos []int32,
	optionPos []int32,
	optionMask []byte,
	targetPos []int32,
	targetMask []byte,
	cardRefPos []int32,
	tokenOverflow []int32,
) {
	tokenIDs = make([]int32, benchBatchSize*benchMaxTokens)
	cuSeqlens = make([]int32, benchBatchSize+1)
	seqLengths = make([]int32, benchBatchSize)
	statePos = make([]int32, benchBatchSize)
	optionPos = make([]int32, benchBatchSize*benchMaxOptions)
	optionMask = make([]byte, benchBatchSize*benchMaxOptions)
	targetPos = make([]int32, benchBatchSize*benchMaxOptions*benchMaxTargets)
	targetMask = make([]byte, benchBatchSize*benchMaxOptions*benchMaxTargets)
	cardRefPos = make([]int32, benchBatchSize*benchMaxCardRefs)
	tokenOverflow = make([]int32, benchBatchSize)
	return
}

func BenchmarkAssembleTokensPackedBatch(b *testing.B) {
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

	plans := make([][]int32, benchBatchSize)
	for i := 0; i < benchBatchSize; i++ {
		plans[i] = benchPlan(i)
	}

	tokenIDs, cuSeqlens, seqLengths, statePos,
		optionPos, optionMask, targetPos, targetMask,
		cardRefPos, tokenOverflow := benchAllocOutputs()

	mt := int64(benchMaxTokens)
	mo := int64(benchMaxOptions)
	mtg := int64(benchMaxTargets)
	mcr := int64(benchMaxCardRefs)

	var totalTokens int64
	for _, p := range plans {
		totalTokens += int64(len(p))
	}
	b.Logf("avg plan length = %d int32s", totalTokens/int64(benchBatchSize))

	b.ReportAllocs()
	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		var packedCursor int32
		cuSeqlens[0] = 0
		for i := 0; i < benchBatchSize; i++ {
			rowStart := int64(packedCursor)
			rowEnd := rowStart + mt
			out := &tokenAssemblerOut{
				tokenIDs:    tokenIDs[rowStart:rowEnd],
				optionPos:   optionPos[int64(i)*mo : int64(i+1)*mo],
				optionMask:  optionMask[int64(i)*mo : int64(i+1)*mo],
				targetPos:   targetPos[int64(i)*mo*mtg : int64(i+1)*mo*mtg],
				targetMask:  targetMask[int64(i)*mo*mtg : int64(i+1)*mo*mtg],
				cardRefPos:  cardRefPos[int64(i)*mcr : int64(i+1)*mcr],
				maxOptions:  benchMaxOptions,
				maxTargets:  benchMaxTargets,
				maxCardRefs: benchMaxCardRefs,
				cursorBase:  packedCursor,
			}
			cursor, overflow, err := assembleTokensFromPlan(plans[i], tables, out, benchMaxTokens)
			if err != nil {
				b.Fatalf("assembleTokensFromPlan row=%d: %v", i, err)
			}
			seqLengths[i] = cursor
			statePos[i] = packedCursor
			cuSeqlens[i+1] = packedCursor + cursor
			if overflow {
				tokenOverflow[i] = 1
			} else {
				tokenOverflow[i] = 0
			}
			packedCursor += cursor
		}
		if iter == 0 {
			b.Logf("packed cursor after batch = %d / %d (overflow rows = %d)",
				packedCursor, benchBatchSize*benchMaxTokens, sumI32(tokenOverflow))
		}
	}
}

func sumI32(s []int32) (n int) {
	for _, v := range s {
		if v != 0 {
			n++
		}
	}
	return
}
