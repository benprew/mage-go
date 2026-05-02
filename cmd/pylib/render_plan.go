package main

import (
	"math"
	"sort"
	"sync"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// renderLifeMin / renderLifeMax bound the OP_LIFE payload range. They must
// match LIFE_MIN / LIFE_MAX in magic_ai/text_encoder/token_tables.py — the
// assembler precomputes a (life, owner)-keyed token table sized to this
// range and rejects out-of-range values.
const (
	renderLifeMin int64 = -30
	renderLifeMax int64 = 300
)

func clampLife(value int64) int32 {
	if value > renderLifeMax {
		return int32(renderLifeMax)
	}
	if value < renderLifeMin {
		return int32(renderLifeMin)
	}
	return int32(value)
}

// renderPlanVersion bumps when an opcode change is not byte-equal to v1.
// v2 adds the “<dict>“ card-body deduplication opcodes (21-24); the v2
// opcodes are additive and only appear when “cfg.dedupCardBodies“ is set,
// so v2 emitters remain byte-equal to v1 until that flag is enabled.
const renderPlanVersion = 2

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
	opOpenDict     // 21: opens the per-snapshot card-body dictionary
	opCloseDict    // 22: closes the dictionary
	opDictEntry    // 23: payload [row]; emit one dict entry (full body)
	opPlaceCardRef // 24: payload [slot, row, status, uuid]; ref to dict entry
	opCount        // 25: payload [N]; emit count[N] span (e.g. <library>{N})
	opStackOpen    // 26: emit shared <stack>
	opStackClose   // 27: emit shared </stack>
	opCommandOpen  // 28: emit shared <command>
	opCommandClose // 29: emit shared </command>
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
	cardID  uuid.UUID
	name    string
	row     int32
	perm    *interactive.PermanentState
}

type renderPlanIndex struct {
	cards      []renderCardRef
	uuidByID   map[uuid.UUID]int32
	rowByID    map[uuid.UUID]int32
	slotByID   map[uuid.UUID]int32
	cardsByKey map[renderZoneKey][]renderCardRef
	// rowOrder lists each unique card cache row that appears in any zone of
	// this snapshot, in deterministic ascending order. Populated for v2 dict
	// emission. Mirrors the Python emitter's ``unique_rows`` (collected in
	// _RENDER_ZONES order, then sorted ascending).
	rowOrder []int32
}

// encodeScratch holds per-call scratch buffers for an encode batch so
// hot-path map/slice allocations are reused across batch rows.
type encodeScratch struct {
	cardIDToSlot  map[string]int64
	renderIndex   renderPlanIndex
	rowSeen       map[int32]struct{}
	tokenPlan     []int32
	tokenPlanLen  [1]int64
	tokenPlanOvf  [1]int64
	directEmitter directTokenEmitter
}

func newEncodeScratch() *encodeScratch {
	return &encodeScratch{
		cardIDToSlot: make(map[string]int64),
		renderIndex: renderPlanIndex{
			uuidByID:   make(map[uuid.UUID]int32),
			rowByID:    make(map[uuid.UUID]int32),
			slotByID:   make(map[uuid.UUID]int32),
			cardsByKey: make(map[renderZoneKey][]renderCardRef),
		},
		rowSeen: make(map[int32]struct{}),
	}
}

func (s *encodeScratch) reset() {
	clear(s.cardIDToSlot)
	idx := &s.renderIndex
	clear(idx.uuidByID)
	clear(idx.rowByID)
	clear(idx.slotByID)
	for k, v := range idx.cardsByKey {
		idx.cardsByKey[k] = v[:0]
	}
	idx.cards = idx.cards[:0]
	idx.rowOrder = idx.rowOrder[:0]
	clear(s.rowSeen)
	s.tokenPlanLen[0] = 0
	s.tokenPlanOvf[0] = 0
}

func (s *encodeScratch) internalRenderPlanView(capacity int64) outputViews {
	if int64(cap(s.tokenPlan)) < capacity {
		s.tokenPlan = make([]int32, capacity)
	}
	s.tokenPlan = s.tokenPlan[:capacity]
	return outputViews{
		renderPlan:         s.tokenPlan,
		renderPlanLengths:  s.tokenPlanLen[:],
		renderPlanOverflow: s.tokenPlanOvf[:],
	}
}

type renderZoneKey struct {
	owner int32
	zone  int32
}

func fillRenderPlan(batchIdx int64, state *apiGameState, pending *apiPending, playerIdx int, cfg encodeConfig, view outputViews, scratch *encodeScratch) *encodeError {
	start := batchIdx * cfg.renderPlanCapacity
	plan := view.renderPlan[start : start+cfg.renderPlanCapacity]
	index := &scratch.renderIndex
	if err := buildRenderPlanIndex(state, playerIdx, index, &scratch.rowSeen); err != nil {
		return err
	}

	w := renderPlanWriter{buf: plan}
	w.write(opOpenState)
	if cfg.dedupCardBodies && len(index.rowOrder) > 0 {
		w.write(opOpenDict)
		for _, row := range index.rowOrder {
			w.write(opDictEntry, row)
		}
		w.write(opCloseDict)
	}
	w.write(opTurn, clampInt32(int64(state.Turn)), int32(indexOrUnknown(stepNames[:], state.Step)))
	emitRenderPlayerScalars(&w, state, playerIdx)
	emitRenderZones(&w, state, playerIdx, *index, cfg)
	emitRenderActions(&w, pending, state, playerIdx, cfg, *index)
	w.write(opCloseState)

	view.renderPlanLengths[batchIdx] = w.cursor
	if w.overflow {
		view.renderPlanOverflow[batchIdx] = 1
	}
	return nil
}

func buildRenderPlanIndex(state *apiGameState, perspectivePlayerIdx int, index *renderPlanIndex, rowSeenPtr *map[int32]struct{}) *encodeError {
	rowSeen := *rowSeenPtr
	// First pass: build the full card lists per (owner, zone) but do NOT
	// assign UUID indices yet. UUID-ordering must match Python's
	// _assign_card_refs which walks zones owner-interleaved (self.bf,
	// opp.bf, self.hand, opp.hand, ...). Doing the UUID pass after the
	// data is collected lets us iterate in that order without changing the
	// per-zone iteration semantics elsewhere.
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		player := renderPlayerState(state, perspectivePlayerIdx, owner)
		if player == nil {
			continue
		}
		for _, zone := range renderZoneOrder {
			key := renderZoneKey{owner: owner, zone: zone}
			cards, err := appendRenderCardsForZone(index.cardsByKey[key][:0], player, owner, zone)
			if err != nil {
				return err
			}
			index.cardsByKey[key] = cards
		}
	}
	// UUID assignment: walk in Python's _ZONE_ORDER (owner-interleaved by
	// zone) so card-ref ids line up byte-for-byte.
	type zoneAssignKey struct {
		owner int32
		zone  int32
	}
	uuidOrder := []zoneAssignKey{
		{renderOwnerSelf, renderZoneBattlefield},
		{renderOwnerOpponent, renderZoneBattlefield},
		{renderOwnerSelf, renderZoneHand},
		{renderOwnerOpponent, renderZoneHand},
		{renderOwnerSelf, renderZoneGraveyard},
		{renderOwnerOpponent, renderZoneGraveyard},
		{renderOwnerSelf, renderZoneExile},
		{renderOwnerOpponent, renderZoneExile},
	}
	for _, key := range uuidOrder {
		cards := index.cardsByKey[renderZoneKey{owner: key.owner, zone: key.zone}]
		for idx := range cards {
			if cards[idx].cardID != uuid.Nil {
				if uuidIdx, ok := index.uuidByID[cards[idx].cardID]; ok {
					cards[idx].uuidIdx = uuidIdx
				} else {
					cards[idx].uuidIdx = int32(len(index.uuidByID))
					index.uuidByID[cards[idx].cardID] = cards[idx].uuidIdx
				}
				index.rowByID[cards[idx].cardID] = cards[idx].row
				index.slotByID[cards[idx].cardID] = cards[idx].slotIdx
			}
			if _, dup := rowSeen[cards[idx].row]; !dup {
				rowSeen[cards[idx].row] = struct{}{}
				index.rowOrder = append(index.rowOrder, cards[idx].row)
			}
			index.cards = append(index.cards, cards[idx])
		}
		// Persist mutations back (cards is a copy of the slice header but
		// shares the backing array, so the uuidIdx writes already landed).
		index.cardsByKey[renderZoneKey{owner: key.owner, zone: key.zone}] = cards
	}
	sort.Slice(index.rowOrder, func(i, j int) bool { return index.rowOrder[i] < index.rowOrder[j] })
	return nil
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

func appendRenderCardsForZone(out []renderCardRef, player *interactive.PlayerState, owner int32, zone int32) ([]renderCardRef, *encodeError) {
	switch zone {
	case renderZoneBattlefield:
		// Take pointers directly into player.Battlefield so each renderCardRef
		// shares the snapshot's PermanentState rather than getting its own
		// heap-allocated copy. The snapshot outlives the index, and the encode
		// path is read-only.
		for idx := range player.Battlefield {
			perm := &player.Battlefield[idx]
			row, ok := cardRowForName(perm.Name)
			if !ok {
				return nil, &encodeError{code: mageEncodeErrEncode, message: "missing card embedding for " + perm.Name}
			}
			out = append(out, renderCardRef{
				zone:    zone,
				owner:   owner,
				slotIdx: renderSlotIndex(owner, zone, idx),
				uuidIdx: -1,
				cardID:  perm.ID,
				name:    perm.Name,
				row:     clampInt32(row),
				perm:    perm,
			})
		}
		return out, nil
	case renderZoneHand:
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
				cardID:  card.ID,
				name:    card.Name,
				row:     clampInt32(row),
			})
		}
		return out, nil
	case renderZoneGraveyard:
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
				cardID:  card.ID,
				name:    card.Name,
				row:     clampInt32(row),
			})
		}
		return out, nil
	default:
		return out, nil
	}
}

func emitRenderPlayerScalars(w *renderPlanWriter, state *apiGameState, playerIdx int) {
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		player := renderPlayerState(state, playerIdx, owner)
		if player == nil {
			continue
		}
		w.write(opLife, owner, clampLife(int64(player.Life)))
		pool := []int{player.ManaPool.White, player.ManaPool.Blue, player.ManaPool.Black, player.ManaPool.Red, player.ManaPool.Green, player.ManaPool.Colorless}
		for colorID, amount := range pool {
			if amount != 0 {
				w.write(opMana, owner, int32(colorID), clampInt32(int64(amount)))
			}
		}
	}
}

// emitRenderZones writes zone blocks in the same order as the Python
// emit_render_plan path:
//
//   - Battlefield  (self, opp)
//   - Hand         (self)            ← opp.hand is fog-of-war redacted
//   - Graveyard    (self, opp)
//   - Exile        (self if non-empty, opp if non-empty)
//   - Library      (self, opp)       ← <{owner}><library>{N}</library></{owner}>
//   - Stack        (shared, single block)
//   - Command      (shared, single block)
//
// Owner-interleave (zone-outer, owner-inner) mirrors Python; the previous
// owner-outer iteration produced a different token order.
func emitRenderZones(w *renderPlanWriter, state *apiGameState, playerIdx int, index renderPlanIndex, cfg encodeConfig) {
	emitCardsForZone := func(owner, zone int32) {
		w.write(opOpenZone, zone, owner)
		for _, card := range index.cardsByKey[renderZoneKey{owner: owner, zone: zone}] {
			if cfg.dedupCardBodies {
				// v2: ref the dict entry, no body splice. Per-card counter /
				// attached_to are skipped to match the Python emitter, which
				// does not emit them in dedup mode.
				w.write(opPlaceCardRef, card.slotIdx, card.row, renderStatusBits(card.perm), card.uuidIdx)
				continue
			}
			w.write(opPlaceCard, card.slotIdx, card.row, renderStatusBits(card.perm), card.uuidIdx)
			if card.perm != nil {
				for ct := core.CounterType(0); ct < core.NumCounters; ct++ {
					count := card.perm.RawCounters[ct]
					if count != 0 {
						w.write(opCounter, int32(ct), int32(count))
					}
				}
				if card.perm.AttachedTo != uuid.Nil {
					targetUUIDIdx := int32(-1)
					if idx, ok := index.uuidByID[card.perm.AttachedTo]; ok {
						targetUUIDIdx = idx
					}
					w.write(opAttachedTo, targetUUIDIdx)
				}
			}
		}
		w.write(opCloseZone)
	}

	// Battlefield, Hand (self only), Graveyard.
	for _, zone := range []int32{renderZoneBattlefield, renderZoneHand, renderZoneGraveyard} {
		for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
			if owner == renderOwnerOpponent && zone == renderZoneHand {
				continue // fog of war
			}
			emitCardsForZone(owner, zone)
		}
	}
	// Exile: skip when empty for that owner.
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		if len(index.cardsByKey[renderZoneKey{owner: owner, zone: renderZoneExile}]) == 0 {
			continue
		}
		emitCardsForZone(owner, renderZoneExile)
	}
	// Library: <{owner}><library>{N}</library></{owner}>. The zone open/close
	// tables already encode <{owner}><library> and </library></{owner}>; the
	// count slots in via opCount.
	for _, owner := range []int32{renderOwnerSelf, renderOwnerOpponent} {
		player := renderPlayerState(state, playerIdx, owner)
		if player == nil {
			continue
		}
		w.write(opOpenZone, renderZoneLibrary, owner)
		w.write(opCount, clampInt32(int64(player.LibraryCount)))
		w.write(opCloseZone)
	}
	// Stack (shared) — always emitted so the model sees the structural slot.
	w.write(opStackOpen)
	w.write(opStackClose)
	// Command zone is omitted in 60-card formats (the engine snapshot does
	// not surface command-zone contents). Re-introduce when commander /
	// conspiracy / emblem support is plumbed through the snapshot API.
	_ = playerIdx
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
	// Permanents always carry the known-tap bit so the assembler knows to
	// emit ``<tapped>`` or ``<untapped>``. Mirrors Python's
	// ``_status_bits_from_card`` which sets STATUS_TAPPED_KNOWN whenever the
	// snapshot reports either Tapped=True or Tapped=False.
	bits := statusTappedKnown
	if perm.Tapped {
		bits |= statusTapped
	}
	if perm.SummonSick {
		bits |= statusSick
	}
	if perm.Attacking {
		bits |= statusAttacking
	}
	if perm.Blocking != uuid.Nil {
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
	if perm.AttachedTo != uuid.Nil {
		bits |= statusIsAttached
	}
	return bits
}

func renderOptionSource(option apiOption, index renderPlanIndex) (int32, int32) {
	for _, id := range []string{option.CardID, option.PermanentID, option.ID} {
		if id == "" {
			continue
		}
		parsed, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		uuidIdx, hasUUID := index.uuidByID[parsed]
		row, hasRow := index.rowByID[parsed]
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

func renderTarget(target apiTarget, selfID uuid.UUID, oppID uuid.UUID, index renderPlanIndex) (int32, int32, int32) {
	if target.ID == "" {
		return -1, -1, renderTargetUnknown
	}
	parsed, err := uuid.Parse(target.ID)
	if err != nil {
		return -1, -1, renderTargetUnknown
	}
	// For player targets the assembler doesn't need a row / uuid index — it
	// emits ``<self>`` or ``<opp>`` directly. Encode the owner index in the
	// row slot (0=self, 1=opp) so the assembler can dispatch on a single
	// payload word without needing to know the player's UUID.
	if parsed == selfID {
		return renderOwnerSelf, -1, renderTargetPlayer
	}
	if parsed == oppID {
		return renderOwnerOpponent, -1, renderTargetPlayer
	}
	uuidIdx, hasUUID := index.uuidByID[parsed]
	row, hasRow := index.rowByID[parsed]
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
