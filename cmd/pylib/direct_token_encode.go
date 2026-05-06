package main

import (
	"time"

	"github.com/google/uuid"
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
	mb := int64(cfg.blankMaxBlanks)
	mv := int64(cfg.blankMaxLegal)

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
	if cfg.blankMaxBlanks > 0 && cfg.blankMaxLegal > 0 {
		inline := classifyDirectInlineOptions(pending, state, playerIdx, index, cfg)
		if err := emitDirectZones(e, state, playerIdx, index, cfg, inline.byCard); err != nil {
			return err
		}
		if err := emitDirectInlineChoices(e, pending, inline, index); err != nil {
			return err
		}
	} else {
		if err := emitDirectZones(e, state, playerIdx, index, cfg, nil); err != nil {
			return err
		}
		if err := emitDirectActions(e, pending, state, playerIdx, cfg, index); err != nil {
			return err
		}
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

func emitDirectZones(e *directTokenEmitter, state *apiGameState, playerIdx int, index renderPlanIndex, cfg encodeConfig, blanksByCard map[uuid.UUID][]inlineBlankOption) error {
	emitCardsForZone := func(owner, zone int32) error {
		e.emitOpenZone(zone, owner)
		for _, card := range index.cardsByZone[zoneOwnerSlot(zone, owner)] {
			if cfg.dedupCardBodies {
				e.emitPlaceCardRef(card.dictSlot, renderStatusBits(card.perm), card.uuidIdx)
				if blanks := blanksByCard[card.cardID]; len(blanks) > 0 {
					for _, blank := range blanks {
						if err := e.emitInlineBlankOption(blank); err != nil {
							return err
						}
					}
				}
				continue
			}
			e.emitPlaceCard(card.row, renderStatusBits(card.perm), card.uuidIdx)
			if blanks := blanksByCard[card.cardID]; len(blanks) > 0 {
				for _, blank := range blanks {
					if err := e.emitInlineBlankOption(blank); err != nil {
						return err
					}
				}
			}
		}
		e.emitCloseZone(zone, owner)
		return nil
	}

	for _, zone := range []int32{renderZoneBattlefield, renderZoneHand, renderZoneGraveyard} {
		for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
			if owner == renderOwnerOpponent && zone == renderZoneHand {
				continue
			}
			if err := emitCardsForZone(owner, zone); err != nil {
				return err
			}
		}
	}
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		if len(index.cardsByZone[zoneOwnerSlot(renderZoneExile, owner)]) == 0 {
			continue
		}
		if err := emitCardsForZone(owner, renderZoneExile); err != nil {
			return err
		}
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
			e.emitPlaceCardRef(card.dictSlot, card.staticStatus, card.uuidIdx)
		} else {
			e.emitPlaceCard(card.row, card.staticStatus, card.uuidIdx)
		}
	}
	e.emitStackClose()
	_ = playerIdx
	return nil
}

func (e *directTokenEmitter) emitInlineBlankOption(blank inlineBlankOption) error {
	if len(blank.legalIDs) > 0 {
		return e.emitBlankLegal(blank.kindID, int32(blank.optIdx), blank.groupKind, blank.legalIDs)
	}
	if err := e.emitBlank(blank.kindID, int32(blank.optIdx)); err != nil {
		return err
	}
	if len(blank.targetLegalIDs) > 0 {
		return e.emitBlankLegal(e.tables.chooseTargetID, int32(blank.optIdx), blankGroupPerBlank, blank.targetLegalIDs)
	}
	return nil
}

func classifyDirectInlineOptions(pending *apiPending, state *apiGameState, playerIdx int, index renderPlanIndex, cfg encodeConfig) inlinePriorityOptions {
	if pending == nil {
		return inlinePriorityOptions{byCard: map[uuid.UUID][]inlineBlankOption{}}
	}
	if pending.Kind == "priority" {
		inline := classifyInlinePriorityOptions(pending)
		enrichInlinePriorityTargetBlanks(&inline, pending, state, playerIdx, index, cfg)
		return inline
	}
	return classifyInlineCombatOptions(pending, state, playerIdx, index, cfg)
}

func enrichInlinePriorityTargetBlanks(inline *inlinePriorityOptions, pending *apiPending, state *apiGameState, playerIdx int, index renderPlanIndex, cfg encodeConfig) {
	if inline == nil || pending == nil || pending.Kind != "priority" {
		return
	}
	selfID, oppID := playerIDs(state, playerIdx)
	for optIdx := range pending.Options {
		option := pending.Options[optIdx]
		if len(option.ValidTargets) == 0 {
			continue
		}
		legalIDs := targetLegalTokenIDs(option.ValidTargets, selfID, oppID, index, cfg)
		if len(legalIDs) == 0 {
			continue
		}
		source := option.CardUUID
		if source == uuid.Nil {
			source = option.PermanentUUID
		}
		blanks := inline.byCard[source]
		for idx := range blanks {
			if blanks[idx].optIdx == optIdx {
				blanks[idx].targetLegalIDs = legalIDs
				break
			}
		}
		inline.byCard[source] = blanks
	}
}

func classifyInlineCombatOptions(pending *apiPending, state *apiGameState, playerIdx int, index renderPlanIndex, cfg encodeConfig) inlinePriorityOptions {
	out := inlinePriorityOptions{byCard: map[uuid.UUID][]inlineBlankOption{}}
	tables := getTokenTables()
	if pending == nil || tables == nil {
		return out
	}
	selfID, oppID := playerIDs(state, playerIdx)
	for optIdx, option := range pending.Options {
		source := option.PermanentUUID
		if source == uuid.Nil {
			source = option.CardUUID
		}
		if source == uuid.Nil {
			continue
		}
		switch pending.Kind {
		case "attackers":
			if entry, ok := index.byCardID[source]; ok && entry.uuidIdx >= 0 && entry.uuidIdx < int32(len(tables.cardRefIDs)) {
				out.byCard[source] = append(out.byCard[source], inlineBlankOption{
					kindID:    tables.chooseTargetID,
					groupKind: blankGroupPerBlank,
					optIdx:    optIdx,
					legalIDs:  []int32{tables.noneID, tables.cardRefIDs[entry.uuidIdx]},
				})
			}
		case "blockers":
			legalIDs := []int32{tables.noneID}
			legalIDs = append(legalIDs, targetLegalTokenIDs(option.ValidTargets, selfID, oppID, index, cfg)...)
			out.byCard[source] = append(out.byCard[source], inlineBlankOption{
				kindID:    tables.chooseBlockID,
				groupKind: blankGroupPerBlank,
				optIdx:    optIdx,
				legalIDs:  legalIDs,
			})
		}
	}
	return out
}

func targetLegalTokenIDs(targets []apiTarget, selfID uuid.UUID, oppID uuid.UUID, index renderPlanIndex, cfg encodeConfig) []int32 {
	tables := getTokenTables()
	if tables == nil {
		return nil
	}
	targetCount := minInt64(int64(len(targets)), int64(cfg.tokenMaxTargets))
	legalIDs := make([]int32, 0, targetCount)
	for tgtIdx := int64(0); tgtIdx < targetCount; tgtIdx++ {
		row, uuidIdx, kind := renderTarget(targets[tgtIdx], selfID, oppID, index)
		switch kind {
		case renderTargetPlayer:
			if row == renderOwnerSelf {
				legalIDs = append(legalIDs, tables.selfID)
			} else if row == renderOwnerOpponent {
				legalIDs = append(legalIDs, tables.oppID)
			}
		case renderTargetPermanent, renderTargetCardInZone:
			if uuidIdx >= 0 && uuidIdx < int32(len(tables.cardRefIDs)) {
				legalIDs = append(legalIDs, tables.cardRefIDs[uuidIdx])
			}
		}
	}
	return legalIDs
}

func emitDirectInlineChoices(e *directTokenEmitter, pending *apiPending, inline inlinePriorityOptions, index renderPlanIndex) error {
	tables := getTokenTables()
	if tables == nil {
		return nil
	}
	passKindID := int32(0)
	if span := tables.actionVerbSpan(0); len(span) > 0 {
		passKindID = span[0]
	}
	for _, optIdx := range inline.passes {
		if err := e.emitBlank(passKindID, int32(optIdx)); err != nil {
			return err
		}
	}
	if pending != nil {
		switch pending.Kind {
		case "permanent", "cards_from_hand", "card_from_library":
			legalIDs := indexedChoiceLegalTokenIDs(pending.Options, index)
			if len(legalIDs) > 0 {
				if err := e.emitBlankLegal(tables.chooseTargetID, -1, blankGroupPerBlank, legalIDs); err != nil {
					return err
				}
			}
		case "may":
			if err := e.emitBlankLegal(tables.chooseMayID, -1, blankGroupPerBlank, []int32{tables.noID, tables.yesID}); err != nil {
				return err
			}
		case "mode":
			if legalIDs := numChoiceLegalTokenIDs(len(pending.Options)); len(legalIDs) > 0 {
				if err := e.emitBlankLegal(tables.chooseModeID, -1, blankGroupPerBlank, legalIDs); err != nil {
					return err
				}
			}
		case "number":
			if legalIDs := numChoiceLegalTokenIDs(len(pending.Options)); len(legalIDs) > 0 {
				if err := e.emitBlankLegal(tables.chooseXDigitID, -1, blankGroupPerBlank, legalIDs); err != nil {
					return err
				}
			}
		case "mana_color":
			legalIDs := manaColorLegalTokenIDs(pending.Options)
			if len(legalIDs) > 0 {
				if err := e.emitBlankLegal(tables.chooseManaSourceID, -1, blankGroupPerBlank, legalIDs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func numChoiceLegalTokenIDs(count int) []int32 {
	tables := getTokenTables()
	if tables == nil || count <= 0 || count > int(tables.numCount) || count > len(tables.numIDs) {
		return nil
	}
	return append([]int32(nil), tables.numIDs[:count]...)
}

func manaColorLegalTokenIDs(options []apiOption) []int32 {
	tables := getTokenTables()
	if tables == nil || tables.manaColorCount < 6 {
		return nil
	}
	out := make([]int32, 0, len(options))
	for _, option := range options {
		colorID, ok := manaColorOptionID(option.Color)
		if !ok {
			return nil
		}
		span := tables.manaGlyphSpan(colorID)
		if len(span) == 0 {
			return nil
		}
		out = append(out, span[0])
	}
	return out
}

func manaColorOptionID(color string) (int32, bool) {
	switch color {
	case "white", "W":
		return 0, true
	case "blue", "U":
		return 1, true
	case "black", "B":
		return 2, true
	case "red", "R":
		return 3, true
	case "green", "G":
		return 4, true
	case "colorless", "C":
		return 5, true
	}
	return 0, false
}

func indexedChoiceLegalTokenIDs(options []apiOption, index renderPlanIndex) []int32 {
	tables := getTokenTables()
	if tables == nil || len(options) == 0 {
		return nil
	}
	cardRefIDs := make([]int32, 0, len(options))
	for _, option := range options {
		if option.IDUUID == uuid.Nil {
			cardRefIDs = cardRefIDs[:0]
			break
		}
		entry, ok := index.byCardID[option.IDUUID]
		if !ok || entry.uuidIdx < 0 || entry.uuidIdx >= int32(len(tables.cardRefIDs)) {
			cardRefIDs = cardRefIDs[:0]
			break
		}
		cardRefIDs = append(cardRefIDs, tables.cardRefIDs[entry.uuidIdx])
	}
	if len(cardRefIDs) == len(options) {
		return cardRefIDs
	}
	return numChoiceLegalTokenIDs(len(options))
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
