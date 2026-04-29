package main

import (
	"math"
	"sort"
	"sync"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

const renderPlanVersion = 1

const (
	opOpenState int32 = iota + 1
	opCloseState
	opTurn
	opLife
	opMana
	opOpenPlayer
	opClosePlayer
	opOpenZone
	opCloseZone
	opPlaceCard
	opCounter
	opAttachedTo
	opOpenActions
	opCloseActions
	opOption
	opTarget
	opLiteralTokens
	opEndCard
	opOpenRawCard
	opCloseRawCard
)

const (
	statusTapped int32 = 1 << iota
	statusSick
	statusAttacking
	statusBlocking
	statusMonstrous
	statusFlipped
	statusFaceDown
	statusPhasedOut
	statusIsCreature
	statusIsLand
	statusIsArtifact
	statusIsAttached
)

const (
	renderOwnerSelf int32 = iota
	renderOwnerOpponent
)

const (
	renderZoneHand int32 = iota
	renderZoneBattlefield
	renderZoneGraveyard
	renderZoneExile
	renderZoneLibrary
	renderZoneStack
	renderZoneCommand
)

const (
	renderTargetPlayer int32 = iota
	renderTargetPermanent
	renderTargetCardInZone
	renderTargetUnknown
)

var renderZoneOrder = [...]int32{
	renderZoneBattlefield,
	renderZoneHand,
	renderZoneGraveyard,
	renderZoneExile,
	renderZoneLibrary,
	renderZoneStack,
	renderZoneCommand,
}

var (
	manaCostRowsOnce sync.Once
	manaCostRows     []string
	manaCostRowByKey map[string]int32
)

type renderPlanWriter struct {
	buf      []int32
	cursor   int64
	overflow bool
}

func (w *renderPlanWriter) write(op int32, args ...int32) {
	width := int64(1 + len(args))
	if w.overflow || w.cursor+width > int64(len(w.buf)) {
		w.overflow = true
		return
	}
	w.buf[w.cursor] = op
	w.cursor++
	copy(w.buf[w.cursor:w.cursor+int64(len(args))], args)
	w.cursor += int64(len(args))
}

type renderCardRef struct {
	zone    int32
	owner   int32
	slotIdx int32
	uuidIdx int32
	id      string
	name    string
	row     int32
	perm    *interactive.PermanentState
}

type renderPlanIndex struct {
	cards      []renderCardRef
	uuidByID   map[string]int32
	rowByID    map[string]int32
	slotByID   map[string]int32
	cardsByKey map[renderZoneKey][]renderCardRef
}

type renderZoneKey struct {
	owner int32
	zone  int32
}

func fillRenderPlan(batchIdx int64, state *apiGameState, pending *apiPending, playerIdx int, cfg encodeConfig, view outputViews) *encodeError {
	start := batchIdx * cfg.renderPlanCapacity
	plan := view.renderPlan[start : start+cfg.renderPlanCapacity]
	index, err := buildRenderPlanIndex(state, playerIdx)
	if err != nil {
		return err
	}

	w := renderPlanWriter{buf: plan}
	w.write(opOpenState)
	w.write(opTurn, clampInt32(int64(state.Turn)), int32(indexOrUnknown(stepNames[:], state.Step)))
	emitRenderPlayerScalars(&w, state, playerIdx)
	emitRenderZones(&w, index)
	emitRenderActions(&w, pending, state, playerIdx, cfg, index)
	w.write(opCloseState)

	view.renderPlanLengths[batchIdx] = w.cursor
	if w.overflow {
		view.renderPlanOverflow[batchIdx] = 1
	}
	return nil
}

func buildRenderPlanIndex(state *apiGameState, perspectivePlayerIdx int) (renderPlanIndex, *encodeError) {
	index := renderPlanIndex{
		uuidByID:   map[string]int32{},
		rowByID:    map[string]int32{},
		slotByID:   map[string]int32{},
		cardsByKey: map[renderZoneKey][]renderCardRef{},
	}
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		player := renderPlayerState(state, perspectivePlayerIdx, owner)
		if player == nil {
			continue
		}
		for _, zone := range renderZoneOrder {
			cards, err := renderCardsForZone(player, owner, zone)
			if err != nil {
				return index, err
			}
			for idx := range cards {
				if cards[idx].id != "" {
					if uuidIdx, ok := index.uuidByID[cards[idx].id]; ok {
						cards[idx].uuidIdx = uuidIdx
					} else {
						cards[idx].uuidIdx = int32(len(index.uuidByID))
						index.uuidByID[cards[idx].id] = cards[idx].uuidIdx
					}
					index.rowByID[cards[idx].id] = cards[idx].row
					index.slotByID[cards[idx].id] = cards[idx].slotIdx
				}
				index.cards = append(index.cards, cards[idx])
			}
			index.cardsByKey[renderZoneKey{owner: owner, zone: zone}] = cards
		}
	}
	return index, nil
}

func renderPlayerState(state *apiGameState, perspectivePlayerIdx int, owner int32) *interactive.PlayerState {
	if len(state.Players) == 0 {
		return nil
	}
	idx := perspectivePlayerIdx
	if owner == renderOwnerOpponent {
		idx = 1 - perspectivePlayerIdx
	}
	if idx < 0 || idx >= len(state.Players) {
		return nil
	}
	return &state.Players[idx]
}

func renderCardsForZone(player *interactive.PlayerState, owner int32, zone int32) ([]renderCardRef, *encodeError) {
	switch zone {
	case renderZoneBattlefield:
		out := make([]renderCardRef, 0, len(player.Battlefield))
		for idx, perm := range player.Battlefield {
			row, ok := cardRowForName(perm.Name)
			if !ok {
				return nil, &encodeError{code: mageEncodeErrEncode, message: "missing card embedding for " + perm.Name}
			}
			perm := perm
			out = append(out, renderCardRef{
				zone:    zone,
				owner:   owner,
				slotIdx: renderSlotIndex(owner, zone, idx),
				uuidIdx: -1,
				id:      perm.ID.String(),
				name:    perm.Name,
				row:     clampInt32(row),
				perm:    &perm,
			})
		}
		return out, nil
	case renderZoneHand:
		out := make([]renderCardRef, 0, len(player.Hand))
		for idx, card := range player.Hand {
			row, ok := cardRowForName(card.Name)
			if !ok {
				return nil, &encodeError{code: mageEncodeErrEncode, message: "missing card embedding for " + card.Name}
			}
			out = append(out, renderCardRef{
				zone:    zone,
				owner:   owner,
				slotIdx: renderSlotIndex(owner, zone, idx),
				uuidIdx: -1,
				id:      card.ID.String(),
				name:    card.Name,
				row:     clampInt32(row),
			})
		}
		return out, nil
	case renderZoneGraveyard:
		out := make([]renderCardRef, 0, len(player.Graveyard))
		for idx, card := range player.Graveyard {
			row, ok := cardRowForName(card.Name)
			if !ok {
				return nil, &encodeError{code: mageEncodeErrEncode, message: "missing card embedding for " + card.Name}
			}
			out = append(out, renderCardRef{
				zone:    zone,
				owner:   owner,
				slotIdx: renderSlotIndex(owner, zone, idx),
				uuidIdx: -1,
				id:      card.ID.String(),
				name:    card.Name,
				row:     clampInt32(row),
			})
		}
		return out, nil
	default:
		return nil, nil
	}
}

func emitRenderPlayerScalars(w *renderPlanWriter, state *apiGameState, playerIdx int) {
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		player := renderPlayerState(state, playerIdx, owner)
		if player == nil {
			continue
		}
		w.write(opLife, owner, clampInt32(int64(player.Life)))
		pool := []int{player.ManaPool.White, player.ManaPool.Blue, player.ManaPool.Black, player.ManaPool.Red, player.ManaPool.Green, player.ManaPool.Colorless}
		for colorID, amount := range pool {
			if amount != 0 {
				w.write(opMana, owner, int32(colorID), clampInt32(int64(amount)))
			}
		}
	}
}

func emitRenderZones(w *renderPlanWriter, index renderPlanIndex) {
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		w.write(opOpenPlayer, owner)
		for _, zone := range renderZoneOrder {
			w.write(opOpenZone, zone, owner)
			for _, card := range index.cardsByKey[renderZoneKey{owner: owner, zone: zone}] {
				w.write(opPlaceCard, card.slotIdx, card.row, renderStatusBits(card.perm), card.uuidIdx)
				if card.perm != nil {
					for ct := core.CounterType(0); ct < core.NumCounters; ct++ {
						count := card.perm.RawCounters[ct]
						if count != 0 {
							w.write(opCounter, int32(ct), int32(count))
						}
					}
					if card.perm.AttachedTo.String() != "" && card.perm.AttachedTo.String() != "00000000-0000-0000-0000-000000000000" {
						targetUUIDIdx := int32(-1)
						if idx, ok := index.uuidByID[card.perm.AttachedTo.String()]; ok {
							targetUUIDIdx = idx
						}
						w.write(opAttachedTo, targetUUIDIdx)
					}
				}
			}
			w.write(opCloseZone)
		}
		w.write(opClosePlayer)
	}
}

func emitRenderActions(w *renderPlanWriter, pending *apiPending, state *apiGameState, playerIdx int, cfg encodeConfig, index renderPlanIndex) {
	w.write(opOpenActions)
	if pending != nil {
		selfID, oppID := playerIDs(state, playerIdx)
		numPresent := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		for optIdx := int64(0); optIdx < numPresent; optIdx++ {
			option := pending.Options[optIdx]
			sourceRow, sourceUUIDIdx := renderOptionSource(option, index)
			w.write(opOption,
				int32(indexOrUnknown(actionKinds[:], option.Kind)),
				sourceRow,
				sourceUUIDIdx,
				manaCostIDForCost(option.ManaCost),
				clampInt32(int64(option.AbilityIndex)),
			)
			for tgtIdx := int64(0); tgtIdx < minInt64(int64(len(option.ValidTargets)), cfg.maxTargetsPerOption); tgtIdx++ {
				row, uuidIdx, kind := renderTarget(option.ValidTargets[tgtIdx], selfID, oppID, index)
				w.write(opTarget, row, uuidIdx, kind)
			}
		}
	}
	w.write(opCloseActions)
}

func renderStatusBits(perm *interactive.PermanentState) int32 {
	if perm == nil {
		return 0
	}
	var bits int32
	if perm.Tapped {
		bits |= statusTapped
	}
	if perm.SummonSick {
		bits |= statusSick
	}
	if perm.Attacking {
		bits |= statusAttacking
	}
	if perm.Blocking.String() != "" && perm.Blocking.String() != "00000000-0000-0000-0000-000000000000" {
		bits |= statusBlocking
	}
	if perm.FaceDown {
		bits |= statusFaceDown
	}
	if perm.PhasedOut {
		bits |= statusPhasedOut
	}
	if perm.IsCreature {
		bits |= statusIsCreature
	}
	if perm.IsLand {
		bits |= statusIsLand
	}
	if perm.IsArtifact {
		bits |= statusIsArtifact
	}
	if perm.AttachedTo.String() != "" && perm.AttachedTo.String() != "00000000-0000-0000-0000-000000000000" {
		bits |= statusIsAttached
	}
	return bits
}

func renderOptionSource(option apiOption, index renderPlanIndex) (int32, int32) {
	for _, id := range []string{option.CardID, option.PermanentID, option.ID} {
		if id == "" {
			continue
		}
		uuidIdx, hasUUID := index.uuidByID[id]
		row, hasRow := index.rowByID[id]
		if hasUUID || hasRow {
			if !hasUUID {
				uuidIdx = -1
			}
			if !hasRow {
				row = -1
			}
			return row, uuidIdx
		}
	}
	if option.CardName != "" {
		row, ok := cardRowForName(option.CardName)
		if ok {
			return clampInt32(row), -1
		}
	}
	return -1, -1
}

func renderTarget(target apiTarget, selfID string, oppID string, index renderPlanIndex) (int32, int32, int32) {
	if target.ID == "" {
		return -1, -1, renderTargetUnknown
	}
	if target.ID == selfID || target.ID == oppID {
		return -1, -1, renderTargetPlayer
	}
	uuidIdx, hasUUID := index.uuidByID[target.ID]
	row, hasRow := index.rowByID[target.ID]
	if hasUUID || hasRow {
		if !hasUUID {
			uuidIdx = -1
		}
		if !hasRow {
			row = -1
		}
		return row, uuidIdx, renderTargetPermanent
	}
	return -1, -1, renderTargetUnknown
}

func renderSlotIndex(owner int32, zone int32, cardIdx int) int32 {
	slotZone := -1
	switch {
	case owner == renderOwnerSelf && zone == renderZoneHand:
		slotZone = 0
	case owner == renderOwnerSelf && zone == renderZoneGraveyard:
		slotZone = 1
	case owner == renderOwnerOpponent && zone == renderZoneGraveyard:
		slotZone = 2
	case owner == renderOwnerSelf && zone == renderZoneBattlefield:
		slotZone = 3
	case owner == renderOwnerOpponent && zone == renderZoneBattlefield:
		slotZone = 4
	default:
		return -1
	}
	if cardIdx >= maxCardsPerZone {
		return -1
	}
	return int32(slotZone*maxCardsPerZone + cardIdx)
}

func registeredManaCostStrings() []string {
	initManaCostRows()
	out := make([]string, len(manaCostRows))
	copy(out, manaCostRows)
	return out
}

func manaCostIDForCost(manaCost string) int32 {
	if manaCost == "" {
		return -1
	}
	initManaCostRows()
	if id, ok := manaCostRowByKey[manaCost]; ok {
		return id
	}
	return -1
}

func initManaCostRows() {
	manaCostRowsOnce.Do(func() {
		seen := map[string]struct{}{}
		for _, name := range mage.RegisteredCardNames() {
			card, err := mage.CreateCard(name)
			if err != nil {
				continue
			}
			cost := card.ManaCost().String()
			if cost != "" {
				seen[cost] = struct{}{}
			}
		}
		manaCostRows = make([]string, 0, len(seen))
		for cost := range seen {
			manaCostRows = append(manaCostRows, cost)
		}
		sort.Strings(manaCostRows)
		manaCostRowByKey = make(map[string]int32, len(manaCostRows))
		for idx, cost := range manaCostRows {
			manaCostRowByKey[cost] = int32(idx)
		}
	})
}

func clampInt32(value int64) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
