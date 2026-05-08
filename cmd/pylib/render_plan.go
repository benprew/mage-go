package main

import (
	"math"
	"slices"
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
	// Inline-blank opcodes (Step 3 of the inline-blank text-encoder
	// migration). EMIT_BLANK writes a kind token at the cursor and primes a
	// per-blank legal-id buffer of size legal_count; EMIT_BLANK_LEGAL appends
	// one legal id to that buffer. legal_count occurrences of EMIT_BLANK_LEGAL
	// must follow each EMIT_BLANK before the next EMIT_BLANK.
	opEmitBlank      // 30: payload [kind_id, group_id, group_kind, legal_count]
	opEmitBlankLegal // 31: payload [token_id]
)

// Inline-blank group-kind enum. Mirrors the Python InlineBlankGroupKind
// values in magic_ai/text_encoder/inline_blanks.py. Stable ints — append-
// only.
const (
	blankGroupPerBlank    int32 = 0
	blankGroupCrossBlank  int32 = 1
	blankGroupConstrained int32 = 2
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
	zone     int32
	owner    int32
	slotIdx  int32
	uuidIdx  int32
	dictSlot int32
	cardID   uuid.UUID
	name     string
	row      int32
	perm     *interactive.PermanentState
	// staticStatus holds status bits for non-permanent cards (e.g.
	// exile face-down). For battlefield cards the bits are computed
	// from PermanentState; for graveyard / hand the value is 0.
	staticStatus int32
}

// cardIDEntry pairs the values previously held in two separate maps
// (uuidByID, rowByID) so the index only does one map write per card and
// renderTarget / renderOptionSource only do one lookup. row=-1 / uuidIdx=-1
// indicate "no value" in the same way the old separate maps did via
// presence.
type cardIDEntry struct {
	uuidIdx int32
	row     int32
}

const renderZoneArrayLen = int(renderZoneCommand) + 1

type renderPlanIndex struct {
	cards    []renderCardRef
	byCardID map[uuid.UUID]cardIDEntry
	// cardsByZone is indexed by zone*2 + owner, where owner is renderOwnerSelf
	// (0) or renderOwnerOpponent (1). Replaces a map[renderZoneKey][]renderCardRef
	// with a fixed array since the keyspace is small (zoneCount*2 == 14) and
	// hit on every card insert + every emitter zone iteration.
	cardsByZone [renderZoneArrayLen * 2][]renderCardRef
	// rowOrder lists each unique card cache row that appears in any zone of
	// this snapshot, in deterministic ascending order. Populated for the
	// legacy row-keyed render-plan dict emission.
	rowOrder []int32
	// dictRowOrder lists each unique row in *insertion order* (the order the
	// card was first seen during index construction). Position in this slice
	// IS the card's per-snapshot dict slot, which the direct emitter uses as
	// the dict-entry token id — so the model can't memorize a stable card
	// identity across snapshots.
	dictRowOrder []int32
	// dictSlotByRow is the sparse half of a sparse-dense int set
	// (Briggs/Torczon). dictSlotByRow[row] is the candidate slot for row;
	// row is actually present iff dictSlotByRow[row] < len(dictRowOrder)
	// and dictRowOrder[dictSlotByRow[row]] == row. This makes reset O(1)
	// (just truncate dictRowOrder) without ever touching dictSlotByRow,
	// and per-row insert / membership is two loads + a compare.
	dictSlotByRow []int32
}

func zoneOwnerSlot(zone, owner int32) int {
	return int(zone)*2 + int(owner)
}

// encodeScratch holds per-call scratch buffers for an encode batch so
// hot-path map/slice allocations are reused across batch rows.
type encodeScratch struct {
	cardIDToSlot     map[string]int64
	renderIndex      renderPlanIndex
	tokenPlan        []int32
	tokenPlanLen     [1]int64
	tokenPlanOvf     [1]int64
	directEmitter    directTokenEmitter
	directOut        tokenAssemblerOut
	blankCollector   blankCollector
	packedOptionPos  []int32
	packedOptionMask []byte
	packedTargetPos  []int32
	packedTargetMask []byte
	// directDirty USED TO live here. The per-row "what slots did the last
	// emit write" state must be associated with the OUTPUT BUFFER, not the
	// scratch — under parallel encode the pool can hand a different scratch
	// to the same row across calls, and the dirty record on a stale scratch
	// no longer reflects what's actually sitting in the buffer. See
	// ``scratchPool.directDirty`` for the per-buffer state.
	// nameRowCache memoizes cardRowForName lookups for the lifetime of
	// the scratch. Names repeat heavily within a snapshot (multiple
	// copies of the same card across battlefield / hand / graveyard /
	// options), so the first lookup pays the lock + map-probe cost and
	// subsequent ones hit a small unlocked map.
	nameRowCache map[string]int32
}

func newEncodeScratch() *encodeScratch {
	// Pre-size the hot-path maps so the typical batch row's ~28 cards
	// don't trigger a rehash during index construction. Sizes are upper
	// bounds for realistic snapshots: 64 distinct UUIDs (battlefield +
	// hand + graveyard for both players) and 64 distinct card rows.
	return &encodeScratch{
		cardIDToSlot: make(map[string]int64, 64),
		renderIndex: renderPlanIndex{
			byCardID: make(map[uuid.UUID]cardIDEntry, 64),
		},
	}
}

func (s *encodeScratch) packedAnchorScratch(
	optionCount int64,
	targetCount int64,
) ([]int32, []byte, []int32, []byte) {
	if optionCount < 0 {
		optionCount = 0
	}
	if targetCount < 0 {
		targetCount = 0
	}
	if int64(cap(s.packedOptionPos)) < optionCount {
		s.packedOptionPos = make([]int32, optionCount)
	}
	if int64(cap(s.packedOptionMask)) < optionCount {
		s.packedOptionMask = make([]byte, optionCount)
	}
	if int64(cap(s.packedTargetPos)) < targetCount {
		s.packedTargetPos = make([]int32, targetCount)
	}
	if int64(cap(s.packedTargetMask)) < targetCount {
		s.packedTargetMask = make([]byte, targetCount)
	}
	return s.packedOptionPos[:optionCount],
		s.packedOptionMask[:optionCount],
		s.packedTargetPos[:targetCount],
		s.packedTargetMask[:targetCount]
}

func (s *encodeScratch) reset() {
	clear(s.cardIDToSlot)
	idx := &s.renderIndex
	clear(idx.byCardID)
	for slot := range idx.cardsByZone {
		idx.cardsByZone[slot] = idx.cardsByZone[slot][:0]
	}
	idx.cards = idx.cards[:0]
	idx.rowOrder = idx.rowOrder[:0]
	// Sparse-dense reset: only truncate the dense side. Stale dictSlotByRow
	// entries are auto-rejected by the membership check, so this stays O(1).
	idx.dictRowOrder = idx.dictRowOrder[:0]
	s.tokenPlanLen[0] = 0
	s.tokenPlanOvf[0] = 0
}

// ensureDictSparse grows dictSlotByRow so it can hold up to rowCount entries.
// Reused across calls — staleness is detected by the dictRowOrder[slot] == row
// check, so no clear is needed when growing.
func (idx *renderPlanIndex) ensureDictSparse(rowCount int32) {
	if int(rowCount) <= len(idx.dictSlotByRow) {
		return
	}
	next := make([]int32, rowCount)
	copy(next, idx.dictSlotByRow)
	idx.dictSlotByRow = next
}

// dictSlotFor returns the per-snapshot slot for row, allocating a fresh one
// (in insertion order) if row hasn't been seen yet. O(1) amortized.
func (idx *renderPlanIndex) dictSlotFor(row int32) int32 {
	if row < 0 || int(row) >= len(idx.dictSlotByRow) {
		return -1
	}
	s := idx.dictSlotByRow[row]
	if s >= 0 && int(s) < len(idx.dictRowOrder) && idx.dictRowOrder[s] == row {
		return s
	}
	s = int32(len(idx.dictRowOrder))
	idx.dictRowOrder = append(idx.dictRowOrder, row)
	idx.dictSlotByRow[row] = s
	return s
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

func fillRenderPlan(batchIdx int64, state *apiGameState, pending *apiPending, playerIdx int, cfg encodeConfig, view outputViews, scratch *encodeScratch) *encodeError {
	start := batchIdx * cfg.renderPlanCapacity
	plan := view.renderPlan[start : start+cfg.renderPlanCapacity]
	if err := buildRenderPlanIndex(state, playerIdx, scratch); err != nil {
		return err
	}
	index := &scratch.renderIndex

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
	if cfg.blankMaxBlanks > 0 && cfg.blankMaxLegal > 0 {
		inline := classifyInlinePriorityOptions(pending)
		emitRenderZones(&w, state, playerIdx, *index, cfg, inline.byCard)
		emitRenderInlineChoices(&w, inline)
	} else {
		emitRenderZones(&w, state, playerIdx, *index, cfg, nil)
		emitRenderActions(&w, pending, state, playerIdx, cfg, *index)
	}
	w.write(opCloseState)

	view.renderPlanLengths[batchIdx] = w.cursor
	if w.overflow {
		view.renderPlanOverflow[batchIdx] = 1
	}
	return nil
}

func buildRenderPlanIndex(state *apiGameState, perspectivePlayerIdx int, scratch *encodeScratch) *encodeError {
	index := &scratch.renderIndex
	if tables := getTokenTables(); tables != nil && tables.cardRowCount > 0 {
		index.ensureDictSparse(tables.cardRowCount)
	}
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
			if zone == renderZoneStack || zone == renderZoneCommand {
				continue
			}
			slot := zoneOwnerSlot(zone, owner)
			cards, err := appendRenderCardsForZone(index.cardsByZone[slot][:0], player, owner, zone, scratch, nil, nil)
			if err != nil {
				return err
			}
			index.cardsByZone[slot] = cards
		}
	}
	stackSlot := zoneOwnerSlot(renderZoneStack, renderOwnerSelf)
	stackCards, err := appendRenderCardsForZone(index.cardsByZone[stackSlot][:0], nil, renderOwnerSelf, renderZoneStack, scratch, state.Stack, state)
	if err != nil {
		return err
	}
	index.cardsByZone[stackSlot] = stackCards
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
		{renderOwnerSelf, renderZoneStack},
	}
	for _, key := range uuidOrder {
		slot := zoneOwnerSlot(key.zone, key.owner)
		cards := index.cardsByZone[slot]
		for idx := range cards {
			if cards[idx].cardID != uuid.Nil {
				if existing, ok := index.byCardID[cards[idx].cardID]; ok {
					cards[idx].uuidIdx = existing.uuidIdx
				} else {
					cards[idx].uuidIdx = int32(len(index.byCardID))
					index.byCardID[cards[idx].cardID] = cardIDEntry{
						uuidIdx: cards[idx].uuidIdx,
						row:     cards[idx].row,
					}
				}
			}
			row := cards[idx].row
			// Sparse-dense dict slot assignment: O(1) amortized membership +
			// insertion. dictRowOrder is the dense insertion-ordered list
			// the direct emitter walks; the per-card dictSlot is the
			// position the card claims in that list.
			cards[idx].dictSlot = index.dictSlotFor(row)
			// Legacy ascending-sorted rowOrder still maintained for the
			// render-plan path, which keys dict ids by row, not slot.
			pos := 0
			for pos < len(index.rowOrder) && index.rowOrder[pos] < row {
				pos++
			}
			if pos == len(index.rowOrder) || index.rowOrder[pos] != row {
				index.rowOrder = append(index.rowOrder, 0)
				copy(index.rowOrder[pos+1:], index.rowOrder[pos:len(index.rowOrder)-1])
				index.rowOrder[pos] = row
			}
			index.cards = append(index.cards, cards[idx])
		}
		// Persist mutations back (cards is a copy of the slice header but
		// shares the backing array, so the uuidIdx writes already landed).
		index.cardsByZone[slot] = cards
	}
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

func appendRenderCardsForZone(
	out []renderCardRef,
	player *interactive.PlayerState,
	owner int32,
	zone int32,
	scratch *encodeScratch,
	stackItems []interactive.StackItemState,
	state *apiGameState,
) ([]renderCardRef, *encodeError) {
	switch zone {
	case renderZoneBattlefield:
		// Take pointers directly into player.Battlefield so each renderCardRef
		// shares the snapshot's PermanentState rather than getting its own
		// heap-allocated copy. The snapshot outlives the index, and the encode
		// path is read-only.
		for idx := range player.Battlefield {
			perm := &player.Battlefield[idx]
			row, ok := scratch.cachedRowForName(perm.Name)
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
				row:     row,
				perm:    perm,
			})
		}
		return out, nil
	case renderZoneHand:
		for idx, card := range player.Hand {
			row, ok := scratch.cachedRowForName(card.Name)
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
				row:     row,
			})
		}
		return out, nil
	case renderZoneGraveyard:
		for idx, card := range player.Graveyard {
			row, ok := scratch.cachedRowForName(card.Name)
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
				row:     row,
			})
		}
		return out, nil
	case renderZoneExile:
		// Face-down exile that the snapshot viewer cannot inspect arrives
		// with Name="" — emit a row=0 sentinel and the face-down status
		// bit so the model sees "card present, identity unknown" instead
		// of being silently dropped (which would break the count signal).
		for idx, card := range player.Exile {
			var row int32
			if card.FaceDown && card.Name == "" {
				row = 0
			} else {
				r, ok := scratch.cachedRowForName(card.Name)
				if !ok {
					return nil, &encodeError{code: mageEncodeErrEncode, message: "missing card embedding for " + card.Name}
				}
				row = r
			}
			ref := renderCardRef{
				zone:    zone,
				owner:   owner,
				slotIdx: renderSlotIndex(owner, zone, idx),
				uuidIdx: -1,
				cardID:  card.ID,
				name:    card.Name,
				row:     row,
			}
			if card.FaceDown {
				ref.staticStatus |= statusFaceDown
			}
			out = append(out, ref)
		}
		return out, nil
	case renderZoneStack:
		for _, item := range stackItems {
			row, name, ok := stackItemCardRow(item, scratch, state)
			if !ok {
				return nil, &encodeError{code: mageEncodeErrEncode, message: "missing card embedding for " + item.Name}
			}
			id := uuid.Nil
			if item.ID != "" {
				if parsed, err := uuid.Parse(item.ID); err == nil {
					id = parsed
				}
			}
			out = append(out, renderCardRef{
				zone:    zone,
				owner:   owner,
				slotIdx: -1,
				uuidIdx: -1,
				cardID:  id,
				name:    name,
				row:     row,
			})
		}
		return out, nil
	default:
		return out, nil
	}
}

func stackItemCardRow(item interactive.StackItemState, scratch *encodeScratch, state *apiGameState) (int32, string, bool) {
	name := item.Name
	if name != "" && name != "Ability" {
		if row, ok := scratch.cachedRowForName(name); ok {
			return row, name, true
		}
	}
	if state != nil && item.ID != "" {
		for _, player := range state.Players {
			for _, perm := range player.Battlefield {
				if perm.ID.String() != item.ID {
					continue
				}
				row, ok := scratch.cachedRowForName(perm.Name)
				return row, perm.Name, ok
			}
		}
	}
	if name == "Ability" {
		return 0, name, true
	}
	row, ok := scratch.cachedRowForName(name)
	return row, name, ok
}

// cachedRowForName memoizes cardRowForName for the lifetime of the
// scratch. Negative cached values mean "lookup failed (missing
// embedding)" so retries don't re-acquire the global RWMutex.
func (s *encodeScratch) cachedRowForName(name string) (int32, bool) {
	if name == "" {
		return 0, true
	}
	if v, ok := s.nameRowCache[name]; ok {
		if v < 0 {
			return 0, false
		}
		return v, true
	}
	row, ok := cardRowForName(name)
	if s.nameRowCache == nil {
		s.nameRowCache = make(map[string]int32, 64)
	}
	if !ok {
		s.nameRowCache[name] = -1
		return 0, false
	}
	clamped := clampInt32(row)
	s.nameRowCache[name] = clamped
	return clamped, true
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
func emitRenderZones(w *renderPlanWriter, state *apiGameState, playerIdx int, index renderPlanIndex, cfg encodeConfig, blanksByCard map[uuid.UUID][]inlineBlankOption) {
	emitCardsForZone := func(owner, zone int32) {
		w.write(opOpenZone, zone, owner)
		for _, card := range index.cardsByZone[zoneOwnerSlot(zone, owner)] {
			status := renderStatusBits(card.perm) | card.staticStatus
			if cfg.dedupCardBodies {
				// v2: ref the dict entry, no body splice. Per-card counter /
				// attached_to are skipped to match the Python emitter, which
				// does not emit them in dedup mode.
				w.write(opPlaceCardRef, card.slotIdx, card.row, status, card.uuidIdx)
				continue
			}
			w.write(opPlaceCard, card.slotIdx, card.row, status, card.uuidIdx)
			if card.perm != nil {
				for ct := range core.NumCounters {
					count := card.perm.RawCounters[ct]
					if count != 0 {
						w.write(opCounter, int32(ct), int32(count))
					}
				}
				if card.perm.AttachedTo != uuid.Nil {
					targetUUIDIdx := int32(-1)
					if entry, ok := index.byCardID[card.perm.AttachedTo]; ok {
						targetUUIDIdx = entry.uuidIdx
					}
					w.write(opAttachedTo, targetUUIDIdx)
				}
			}
			if blanks := blanksByCard[card.cardID]; len(blanks) > 0 {
				for _, blank := range blanks {
					emitInlineBlank(w, blank.kindID)
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
		if len(index.cardsByZone[zoneOwnerSlot(renderZoneExile, owner)]) == 0 {
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

type inlineBlankOption struct {
	kindID         int32
	groupKind      int32
	abilityIdx     int
	id             string
	optIdx         int
	legalIDs       []int32
	targetLegalIDs []int32
}

type inlinePriorityOptions struct {
	byCard map[uuid.UUID][]inlineBlankOption
	passes []int
}

func classifyInlinePriorityOptions(pending *apiPending) inlinePriorityOptions {
	out := inlinePriorityOptions{byCard: map[uuid.UUID][]inlineBlankOption{}}
	tables := getTokenTables()
	if pending == nil || tables == nil {
		return out
	}
	passKindID := int32(0)
	if span := tables.actionVerbSpan(0); len(span) > 0 {
		passKindID = span[0]
	}
	for optIdx, option := range pending.Options {
		switch option.Kind {
		case "play_land", "cast_spell", "play", "cast":
			source := option.CardUUID
			if source == uuid.Nil {
				source = option.PermanentUUID
			}
			if source != uuid.Nil {
				out.byCard[source] = append(out.byCard[source], inlineBlankOption{
					kindID:     tables.choosePlayID,
					groupKind:  blankGroupCrossBlank,
					abilityIdx: option.AbilityIndex,
					id:         option.ID,
					optIdx:     optIdx,
				})
			}
		case "activate_ability", "activate", "activated_ability":
			source := option.PermanentUUID
			if source == uuid.Nil {
				source = option.CardUUID
			}
			if source != uuid.Nil {
				out.byCard[source] = append(out.byCard[source], inlineBlankOption{
					kindID:     tables.useAbilityID,
					groupKind:  blankGroupCrossBlank,
					abilityIdx: option.AbilityIndex,
					id:         option.ID,
					optIdx:     optIdx,
				})
			}
		case "pass":
			out.passes = append(out.passes, optIdx)
			_ = passKindID
		}
	}
	for cardID := range out.byCard {
		sort.Slice(out.byCard[cardID], func(i, j int) bool {
			a := out.byCard[cardID][i]
			b := out.byCard[cardID][j]
			if a.kindID != b.kindID {
				return a.kindID < b.kindID
			}
			if a.abilityIdx != b.abilityIdx {
				return a.abilityIdx < b.abilityIdx
			}
			if a.id != b.id {
				return a.id < b.id
			}
			return a.optIdx < b.optIdx
		})
	}
	return out
}

func emitInlineBlank(w *renderPlanWriter, kindID int32) {
	tables := getTokenTables()
	if tables == nil {
		return
	}
	w.write(opEmitBlank, kindID, 0, blankGroupCrossBlank, 1)
	w.write(opEmitBlankLegal, tables.chosenID)
}

func emitRenderInlineChoices(w *renderPlanWriter, inline inlinePriorityOptions) {
	tables := getTokenTables()
	if tables == nil {
		return
	}
	passKindID := int32(0)
	if span := tables.actionVerbSpan(0); len(span) > 0 {
		passKindID = span[0]
	}
	for range inline.passes {
		emitInlineBlank(w, passKindID)
	}
}

func emitRenderActions(w *renderPlanWriter, pending *apiPending, state *apiGameState, playerIdx int, cfg encodeConfig, index renderPlanIndex) {
	w.write(opOpenActions)
	if pending != nil {
		selfID, oppID := playerIDs(state, playerIdx)
		numPresent := minInt64(int64(len(pending.Options)), cfg.maxOptions)
		for optIdx := range numPresent {
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
	// In practice an option carries exactly one source identifier:
	// cast_spell / play_land / choice → CardUUID, the rest → PermanentUUID
	// (with ChoiceMay using IDUUID). Try the most likely field first per
	// kind, then fall back through the remaining ones for safety.
	var first uuid.UUID
	switch option.Kind {
	case "cast_spell", "play_land", "choice":
		first = option.CardUUID
	default:
		first = option.PermanentUUID
	}
	if first != uuid.Nil {
		if entry, ok := index.byCardID[first]; ok {
			return entry.row, entry.uuidIdx
		}
	}
	if option.CardUUID != uuid.Nil && option.CardUUID != first {
		if entry, ok := index.byCardID[option.CardUUID]; ok {
			return entry.row, entry.uuidIdx
		}
	}
	if option.PermanentUUID != uuid.Nil && option.PermanentUUID != first {
		if entry, ok := index.byCardID[option.PermanentUUID]; ok {
			return entry.row, entry.uuidIdx
		}
	}
	if option.IDUUID != uuid.Nil {
		if entry, ok := index.byCardID[option.IDUUID]; ok {
			return entry.row, entry.uuidIdx
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
	if target.IDUUID == uuid.Nil {
		return -1, -1, renderTargetUnknown
	}
	// For player targets the assembler doesn't need a row / uuid index — it
	// emits ``<self>`` or ``<opp>`` directly. Encode the owner index in the
	// row slot (0=self, 1=opp) so the assembler can dispatch on a single
	// payload word without needing to know the player's UUID.
	if target.IDUUID == selfID {
		return renderOwnerSelf, -1, renderTargetPlayer
	}
	if target.IDUUID == oppID {
		return renderOwnerOpponent, -1, renderTargetPlayer
	}
	if entry, ok := index.byCardID[target.IDUUID]; ok {
		return entry.row, entry.uuidIdx, renderTargetPermanent
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
		slices.Sort(manaCostRows)
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
