package main

import (
	"time"
)

func fillTokenAssemblyDirectPacked(
	outputBatchIdx int64,
	packedCursor int32,
	state *apiGameState,
	pending *apiPending,
	playerIdx int,
	cfg encodeConfig,
	outputView outputViews,
	scratch *encodeScratch,
	dirty *directDirtyState,
) (int32, time.Duration, *encodeError) {
	tables := getTokenTables()
	if tables == nil {
		return packedCursor, 0, &encodeError{
			code:    mageEncodeErrEncodeFailure,
			message: "MageRegisterTokenTables must be called before MageEncodeTokensPacked",
		}
	}
	if err := buildRenderPlanIndex(state, playerIdx, scratch); err != nil {
		return packedCursor, 0, err
	}
	index := &scratch.renderIndex

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

	out := &scratch.directOut
	*out = tokenAssemblerOut{
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

	outputView.packedSeqLengths[outputBatchIdx] = 0
	outputView.packedStatePositions[outputBatchIdx] = 0
	outputView.packedCuSeqlens[outputBatchIdx+1] = packedCursor
	outputView.packedTokenOverflow[outputBatchIdx] = 0

	emitter := &scratch.directEmitter
	emitter.reset(tables, out, cfg.tokenMaxTokens, dirty)
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
	if cfg.dedupCardBodies && len(index.dictRowOrder) > 0 {
		e.emitOpenDict()
		for slot, row := range index.dictRowOrder {
			e.emitDictEntry(int32(slot), row)
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
	_ = pending
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
		for _, card := range index.cardsByZone[zoneOwnerSlot(zone, owner)] {
			if cfg.dedupCardBodies {
				e.emitPlaceCardRef(card.dictSlot, card.row, renderStatusBits(card.perm), card.uuidIdx)
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
		if len(index.cardsByZone[zoneOwnerSlot(renderZoneExile, owner)]) == 0 {
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
	for _, card := range index.cardsByZone[zoneOwnerSlot(renderZoneStack, renderOwnerSelf)] {
		if cfg.dedupCardBodies {
			e.emitPlaceCardRef(card.dictSlot, card.row, card.staticStatus, card.uuidIdx)
		} else {
			e.emitPlaceCard(card.row, card.staticStatus, card.uuidIdx)
		}
	}
	e.emitStackClose()
	return nil
}
