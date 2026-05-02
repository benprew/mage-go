package main

import "time"

func fillTokenAssemblyDirectPacked(
	outputBatchIdx int64,
	packedCursor int32,
	state *apiGameState,
	pending *apiPending,
	playerIdx int,
	cfg encodeConfig,
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
	index := &scratch.renderIndex
	if err := buildRenderPlanIndex(state, playerIdx, index, &scratch.rowSeen); err != nil {
		return packedCursor, 0, err
	}

	mt := int64(cfg.tokenMaxTokens)
	mo := int64(cfg.tokenMaxOptions)
	mtg := int64(cfg.tokenMaxTargets)
	mcr := int64(cfg.tokenMaxCardRefs)

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
		optionPos:   outputView.packedOptionPos[outputBatchIdx*mo : (outputBatchIdx+1)*mo],
		optionMask:  outputView.packedOptionMask[outputBatchIdx*mo : (outputBatchIdx+1)*mo],
		targetPos:   outputView.packedTargetPos[outputBatchIdx*mo*mtg : (outputBatchIdx+1)*mo*mtg],
		targetMask:  outputView.packedTargetMask[outputBatchIdx*mo*mtg : (outputBatchIdx+1)*mo*mtg],
		cardRefPos:  outputView.packedCardRefPos[outputBatchIdx*mcr : (outputBatchIdx+1)*mcr],
		maxOptions:  cfg.tokenMaxOptions,
		maxTargets:  cfg.tokenMaxTargets,
		maxCardRefs: cfg.tokenMaxCardRefs,
		cursorBase:  packedCursor,
	}

	outputView.packedSeqLengths[outputBatchIdx] = 0
	outputView.packedStatePositions[outputBatchIdx] = 0
	outputView.packedCuSeqlens[outputBatchIdx+1] = packedCursor
	outputView.packedTokenOverflow[outputBatchIdx] = 0

	emitter := &scratch.directEmitter
	emitter.reset(tables, out, cfg.tokenMaxTokens)
	if err := emitDirectTokens(emitter, state, pending, playerIdx, cfg, *index); err != nil {
		return packedCursor, 0, &encodeError{
			code:    mageEncodeErrEncodeFailure,
			message: err.Error(),
		}
	}
	cursor, overflow := emitter.finish()

	metadataStart := time.Now()
	outputView.packedSeqLengths[outputBatchIdx] = cursor
	outputView.packedStatePositions[outputBatchIdx] = packedCursor
	outputView.packedCuSeqlens[outputBatchIdx+1] = packedCursor + cursor
	if overflow {
		outputView.packedTokenOverflow[outputBatchIdx] = 1
	}
	return packedCursor + cursor, time.Since(metadataStart), nil
}

func emitDirectTokens(
	e *directTokenEmitter,
	state *apiGameState,
	pending *apiPending,
	playerIdx int,
	cfg encodeConfig,
	index renderPlanIndex,
) error {
	e.emitOpenState()
	if cfg.dedupCardBodies && len(index.rowOrder) > 0 {
		e.emitOpenDict()
		for _, row := range index.rowOrder {
			e.emitDictEntry(row)
		}
		e.emitCloseDict()
	}
	if err := e.emitTurn(clampInt32(int64(state.Turn)), int32(indexOrUnknown(stepNames[:], state.Step))); err != nil {
		return err
	}
	if err := emitDirectPlayerScalars(e, state, playerIdx); err != nil {
		return err
	}
	if err := emitDirectZones(e, state, playerIdx, index, cfg); err != nil {
		return err
	}
	if err := emitDirectActions(e, pending, state, playerIdx, cfg, index); err != nil {
		return err
	}
	e.emitCloseState()
	return nil
}

func emitDirectPlayerScalars(e *directTokenEmitter, state *apiGameState, playerIdx int) error {
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		player := renderPlayerState(state, playerIdx, owner)
		if player == nil {
			continue
		}
		if err := e.emitLife(owner, clampLife(int64(player.Life))); err != nil {
			return err
		}
		pool := []int{player.ManaPool.White, player.ManaPool.Blue, player.ManaPool.Black, player.ManaPool.Red, player.ManaPool.Green, player.ManaPool.Colorless}
		for colorID, amount := range pool {
			if amount != 0 {
				e.emitMana(owner, int32(colorID), clampInt32(int64(amount)))
			}
		}
	}
	return nil
}

func emitDirectZones(e *directTokenEmitter, state *apiGameState, playerIdx int, index renderPlanIndex, cfg encodeConfig) error {
	emitCardsForZone := func(owner, zone int32) {
		e.emitOpenZone(zone, owner)
		for _, card := range index.cardsByKey[renderZoneKey{owner: owner, zone: zone}] {
			if cfg.dedupCardBodies {
				e.emitPlaceCardRef(card.row, renderStatusBits(card.perm), card.uuidIdx)
				continue
			}
			e.emitPlaceCard(card.row, renderStatusBits(card.perm), card.uuidIdx)
		}
		e.emitCloseZone(zone, owner)
	}

	for _, zone := range []int32{renderZoneBattlefield, renderZoneHand, renderZoneGraveyard} {
		for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
			if owner == renderOwnerOpponent && zone == renderZoneHand {
				continue
			}
			emitCardsForZone(owner, zone)
		}
	}
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		if len(index.cardsByKey[renderZoneKey{owner: owner, zone: renderZoneExile}]) == 0 {
			continue
		}
		emitCardsForZone(owner, renderZoneExile)
	}
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		player := renderPlayerState(state, playerIdx, owner)
		if player == nil {
			continue
		}
		e.emitOpenZone(renderZoneLibrary, owner)
		if err := e.emitCount(clampInt32(int64(player.LibraryCount))); err != nil {
			return err
		}
		e.emitCloseZone(renderZoneLibrary, owner)
	}
	e.emitStackOpen()
	e.emitStackClose()
	_ = playerIdx
	return nil
}

func emitDirectActions(e *directTokenEmitter, pending *apiPending, state *apiGameState, playerIdx int, cfg encodeConfig, index renderPlanIndex) error {
	e.emitOpenActions()
	if pending != nil {
		selfID, oppID := playerIDs(state, playerIdx)
		numPresent := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		for optIdx := int64(0); optIdx < numPresent; optIdx++ {
			option := pending.Options[optIdx]
			sourceRow, sourceUUIDIdx := renderOptionSource(option, index)
			if err := e.emitOption(
				int32(indexOrUnknown(actionKinds[:], option.Kind)),
				sourceRow,
				sourceUUIDIdx,
				clampInt32(int64(option.AbilityIndex)),
			); err != nil {
				return err
			}
			for tgtIdx := int64(0); tgtIdx < minInt64(int64(len(option.ValidTargets)), cfg.maxTargetsPerOption); tgtIdx++ {
				row, uuidIdx, kind := renderTarget(option.ValidTargets[tgtIdx], selfID, oppID, index)
				e.emitTarget(row, uuidIdx, kind)
			}
		}
	}
	e.emitCloseActions()
	return nil
}
