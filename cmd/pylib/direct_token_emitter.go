package main

import (
	"fmt"
	"math/bits"
)

// directDirtyState records exactly what slots a previous run of the direct
// emitter wrote into a shared output row, so the next reset only zeroes
// those slots instead of the full per-row capacity. Lives on encodeScratch
// (one entry per batch row) so dirty info persists across calls that reuse
// the same scratch.
type directDirtyState struct {
	// optionWatermark is the count of option_pos / option_mask slots that
	// were written by the last reset of this row. Slots [0, watermark) need
	// re-zeroing; slots beyond are still pristine sentinels from the
	// caller's allocation.
	optionWatermark int32
	// targetWatermark is the highest target slot index + 1 written by the
	// last reset (i.e. the smallest prefix of target_pos / target_mask that
	// covers all the dirty slots).
	targetWatermark int32
	// cardRefSeen doubles as the live "have we seen this card-ref idx yet"
	// bitmap during emission AND, after the row finishes, as the dirty
	// record for which cardRefPos slots need re-zeroing next time. Bit i
	// set => card ref i was emitted this row.
	cardRefSeen [tokenAssemblerMaxCardRefs / 64]uint64
	// initialized is false until the first reset. While false, reset does a
	// full clear (caller's pre-init may not have laid down sentinels).
	initialized bool
}

type directTokenEmitter struct {
	tables          *tokenTables
	out             *tokenAssemblerOut
	dirty           *directDirtyState
	maxTokens       int32
	cursor          int32
	overflow        bool
	nextOption      int32
	maxTargetSlot   int32 // highest written target_pos index + 1, in row-local coords
	curOptionIdx    int32
	curTargetCount  int32
	optionOpen      bool
	scalarOwnerOpen int32
}

func newDirectTokenEmitter(tables *tokenTables, out *tokenAssemblerOut, maxTokens int32, dirty *directDirtyState) *directTokenEmitter {
	e := &directTokenEmitter{}
	e.reset(tables, out, maxTokens, dirty)
	return e
}

// reset re-binds an emitter to a new output row and clears the per-row
// state. Uses the row's directDirtyState to clear ONLY slots dirtied by
// the previous run on this row: clear() (memclr) for the mask arrays,
// fill loops bounded by the watermark for the -1 sentinels, and a
// trailing-zeros walk over the bitset for the per-card-ref positions.
//
// First-time use of a row (initialized=false) does a full clear because
// the caller's allocator may not have laid down sentinels. After that the
// per-row dirty state caps the work.
func (e *directTokenEmitter) reset(tables *tokenTables, out *tokenAssemblerOut, maxTokens int32, dirty *directDirtyState) {
	if !dirty.initialized {
		fillInt32(out.optionPos, -1)
		clear(out.optionMask)
		fillInt32(out.targetPos, -1)
		clear(out.targetMask)
		fillInt32(out.cardRefPos, -1)
		dirty.initialized = true
	} else {
		if w := dirty.optionWatermark; w > 0 {
			if int(w) > len(out.optionPos) {
				w = int32(len(out.optionPos))
			}
			fillInt32(out.optionPos[:w], -1)
			clear(out.optionMask[:w])
		}
		if w := dirty.targetWatermark; w > 0 {
			if int(w) > len(out.targetPos) {
				w = int32(len(out.targetPos))
			}
			fillInt32(out.targetPos[:w], -1)
			clear(out.targetMask[:w])
		}
		// Walk the cardRefSeen bitset and reset only those positions.
		for word, m := range dirty.cardRefSeen {
			for m != 0 {
				bit := bits.TrailingZeros64(m)
				idx := word*64 + bit
				if idx < len(out.cardRefPos) {
					out.cardRefPos[idx] = -1
				}
				m &= m - 1
			}
		}
	}

	clear(dirty.cardRefSeen[:])
	dirty.optionWatermark = 0
	dirty.targetWatermark = 0

	e.tables = tables
	e.out = out
	e.dirty = dirty
	e.maxTokens = maxTokens
	e.cursor = 0
	e.overflow = false
	e.nextOption = 0
	e.maxTargetSlot = 0
	e.curOptionIdx = -1
	e.curTargetCount = 0
	e.optionOpen = false
	e.scalarOwnerOpen = -1
}

func (e *directTokenEmitter) writeSpan(span []int32) {
	if e.overflow || span == nil {
		return
	}
	n := int32(len(span))
	if e.cursor+n > e.maxTokens {
		room := e.maxTokens - e.cursor
		if room > 0 {
			copy(e.out.tokenIDs[e.cursor:e.cursor+room], span[:room])
			e.cursor += room
		}
		e.overflow = true
		return
	}
	copy(e.out.tokenIDs[e.cursor:e.cursor+n], span)
	e.cursor += n
}

func (e *directTokenEmitter) writeSingle(id int32) int32 {
	if e.overflow {
		return -1
	}
	if e.cursor >= e.maxTokens {
		e.overflow = true
		return -1
	}
	pos := e.cursor
	e.out.tokenIDs[e.cursor] = id
	e.cursor++
	return pos
}

func (e *directTokenEmitter) emitFragment(fragID int32) {
	e.writeSpan(e.tables.fragmentSpan(fragID))
}

func (e *directTokenEmitter) emitCardRef(uuidIdx int32) bool {
	if uuidIdx < 0 || uuidIdx >= tokenAssemblerMaxCardRefs || uuidIdx >= e.tables.cardRefCount {
		return false
	}
	pos := e.writeSingle(e.tables.cardRefIDs[uuidIdx])
	if pos < 0 {
		return false
	}
	mask := uint64(1) << (uint32(uuidIdx) & 63)
	word := uint32(uuidIdx) >> 6
	if e.dirty.cardRefSeen[word]&mask == 0 {
		e.dirty.cardRefSeen[word] |= mask
		e.out.cardRefPos[uuidIdx] = pos + e.out.cursorBase
	}
	return true
}

func (e *directTokenEmitter) closeScalarOwner() {
	if e.scalarOwnerOpen < 0 {
		return
	}
	if e.scalarOwnerOpen == renderOwnerSelf {
		e.emitFragment(fragCloseSelf)
	} else {
		e.emitFragment(fragCloseOpp)
	}
	e.scalarOwnerOpen = -1
}

func (e *directTokenEmitter) closeOption() {
	if e.optionOpen {
		e.emitFragment(fragCloseOption)
		e.optionOpen = false
	}
}

func (e *directTokenEmitter) emitOpenState() {
	e.emitFragment(fragBosState)
}

func (e *directTokenEmitter) emitCloseState() {
	e.closeScalarOwner()
	e.closeOption()
	e.emitFragment(fragCloseStateEos)
}

func (e *directTokenEmitter) emitTurn(turn, stepID int32) error {
	e.closeScalarOwner()
	if stepID < 0 || stepID >= e.tables.stepCount {
		stepID = e.tables.stepCount - 1
	}
	span := e.tables.turnStepSpan(turn, stepID)
	if span == nil {
		return fmt.Errorf(
			"OP_TURN out of bounds: turn=%d step=%d (range %d..%d)",
			turn, stepID, e.tables.turnMin, e.tables.turnMax,
		)
	}
	e.writeSpan(span)
	return nil
}

func (e *directTokenEmitter) emitLife(owner, life int32) error {
	e.closeScalarOwner()
	span := e.tables.lifeOwnerSpan(life, owner)
	if span == nil {
		return fmt.Errorf(
			"OP_LIFE out of bounds: life=%d owner=%d (range %d..%d)",
			life, owner, e.tables.lifeMin, e.tables.lifeMax,
		)
	}
	e.writeSpan(span)
	e.scalarOwnerOpen = owner
	return nil
}

func (e *directTokenEmitter) emitMana(owner, colorID, amount int32) {
	if e.scalarOwnerOpen >= 0 && e.scalarOwnerOpen != owner {
		e.closeScalarOwner()
	}
	if e.scalarOwnerOpen < 0 {
		if owner == renderOwnerSelf {
			e.emitFragment(fragSelfMana)
		} else {
			e.emitFragment(fragOppMana)
		}
		e.scalarOwnerOpen = owner
	}
	if colorID >= 0 && colorID < e.tables.manaColorCount && amount > 0 {
		glyph := e.tables.manaGlyphSpan(colorID)
		for r := int32(0); r < amount && !e.overflow; r++ {
			e.writeSpan(glyph)
		}
	}
}

func (e *directTokenEmitter) emitOpenZone(zone, owner int32) {
	e.closeScalarOwner()
	e.closeOption()
	e.writeSpan(e.tables.zoneOpenSpan(zone, owner))
}

func (e *directTokenEmitter) emitCloseZone(zone, owner int32) {
	e.closeScalarOwner()
	e.closeOption()
	e.writeSpan(e.tables.zoneCloseSpan(zone, owner))
}

func (e *directTokenEmitter) emitOpenActions() {
	e.closeScalarOwner()
	e.emitFragment(fragOpenActions)
}

func (e *directTokenEmitter) emitCloseActions() {
	e.closeScalarOwner()
	e.closeOption()
	e.emitFragment(fragCloseActions)
}

func (e *directTokenEmitter) emitOption(kindID, sourceRow, sourceUUIDIdx, abilityIdx int32) error {
	e.closeScalarOwner()
	e.closeOption()
	pos := e.writeSingle(e.tables.optionID)
	if pos >= 0 && e.nextOption < e.out.maxOptions {
		e.out.optionPos[e.nextOption] = pos + e.out.cursorBase
		e.out.optionMask[e.nextOption] = 1
		e.curOptionIdx = e.nextOption
		e.curTargetCount = 0
		e.nextOption++
	}
	e.optionOpen = true

	verbSpan := e.tables.actionVerbSpan(kindID)
	kindKnown := verbSpan != nil
	var flags uint8
	if kindID >= 0 && int(kindID) < len(kindFlags) {
		flags = kindFlags[kindID]
	}
	if kindKnown {
		e.writeSpan(verbSpan)
	}
	if kindKnown && flags&kindFlagHasSource != 0 {
		if !e.emitCardRef(sourceUUIDIdx) {
			if sourceRow >= 0 && sourceRow < e.tables.cardRowCount {
				e.writeSpan(e.tables.cardNameSpan(sourceRow))
			}
		}
	}
	if abilityIdx >= 0 && flags&kindFlagAbility != 0 {
		span := e.tables.abilitySpan(abilityIdx)
		if span == nil {
			return fmt.Errorf(
				"OP_OPTION out of bounds: ability=%d (range %d..%d)",
				abilityIdx, e.tables.abilityMin, e.tables.abilityMax,
			)
		}
		e.writeSpan(span)
	}
	return nil
}

func (e *directTokenEmitter) emitTarget(targetRow, targetUUIDIdx, targetKind int32) {
	e.closeScalarOwner()
	pos := e.writeSingle(e.tables.targetOpenID)
	if pos >= 0 && e.curOptionIdx >= 0 && e.curTargetCount < e.out.maxTargets {
		idx := e.curOptionIdx*e.out.maxTargets + e.curTargetCount
		e.out.targetPos[idx] = pos + e.out.cursorBase
		e.out.targetMask[idx] = 1
		e.curTargetCount++
		if idx+1 > e.maxTargetSlot {
			e.maxTargetSlot = idx + 1
		}
	}
	switch targetKind {
	case renderTargetPlayer:
		if targetRow == renderOwnerSelf {
			e.writeSingle(e.tables.selfID)
		} else {
			e.writeSingle(e.tables.oppID)
		}
	default:
		if !e.emitCardRef(targetUUIDIdx) {
			if targetRow >= 0 && targetRow < e.tables.cardRowCount {
				e.writeSpan(e.tables.cardNameSpan(targetRow))
			} else {
				e.emitFragment(fragTargetFallback)
			}
		}
	}
	e.writeSingle(e.tables.targetCloseID)
}

func (e *directTokenEmitter) emitCount(amount int32) error {
	e.closeScalarOwner()
	span := e.tables.countSpan(amount)
	if span == nil {
		return fmt.Errorf(
			"OP_COUNT out of bounds: amount=%d (range %d..%d)",
			amount, e.tables.countMin, e.tables.countMax,
		)
	}
	e.writeSpan(span)
	return nil
}

func (e *directTokenEmitter) emitPlaceCard(row, status, uuidIdx int32) {
	e.closeScalarOwner()
	e.emitCardRef(uuidIdx)
	if row >= 0 && row < e.tables.cardRowCount {
		e.writeSpan(e.tables.cardBodySpan(row))
	}
	e.emitStatus(status)
	e.writeSpan(e.tables.cardCloser)
}

// emitPlaceCardRef writes a placement that points back at the dict prologue.
// Prefer per-snapshot slot ids when present; older Python token-table ABIs
// only provide row-keyed dict-entry ids, so fall back to those.
func (e *directTokenEmitter) emitPlaceCardRef(slot, row, status, uuidIdx int32) {
	e.closeScalarOwner()
	e.emitCardRef(uuidIdx)
	e.writeSingle(e.tables.cardOpenID)
	if slot >= 0 && slot < int32(len(e.tables.dictSlotIDs)) {
		e.writeSingle(e.tables.dictSlotIDs[slot])
	} else if row >= 0 && row < int32(len(e.tables.dictEntryIDs)) {
		e.writeSingle(e.tables.dictEntryIDs[row])
	}
	e.emitStatus(status)
	e.writeSpan(e.tables.cardCloser)
}

func (e *directTokenEmitter) emitStatus(status int32) {
	if status&statusTappedKnown != 0 {
		if status&statusTapped != 0 {
			e.writeSpan(e.tables.statusTapped)
		} else {
			e.writeSpan(e.tables.statusUntapped)
		}
	} else if status&statusTapped != 0 {
		e.writeSpan(e.tables.statusTapped)
	}
}

func (e *directTokenEmitter) emitBlank(kindID, optionIndex int32) error {
	tables := e.tables
	if tables == nil {
		return nil
	}
	return e.emitBlankLegal(kindID, optionIndex, blankGroupCrossBlank, []int32{tables.chosenID})
}

func (e *directTokenEmitter) emitBlankLegal(kindID, optionIndex, groupKind int32, legalIDs []int32) error {
	tables := e.tables
	if tables == nil || len(legalIDs) == 0 {
		return nil
	}
	pos := e.writeSingle(kindID)
	if pos < 0 || e.out.blank == nil {
		return nil
	}
	if err := e.out.blank.recordBlank(
		pos+e.out.cursorBase,
		kindID,
		0,
		groupKind,
		optionIndex,
		int32(len(legalIDs)),
	); err != nil {
		return err
	}
	for _, legalID := range legalIDs {
		if err := e.out.blank.recordLegal(legalID); err != nil {
			return err
		}
	}
	return nil
}

func (e *directTokenEmitter) emitOpenDict() {
	e.closeScalarOwner()
	e.writeSingle(e.tables.dictOpenID)
}

func (e *directTokenEmitter) emitCloseDict() {
	e.closeScalarOwner()
	e.writeSingle(e.tables.dictCloseID)
}

// emitDictEntry writes one entry in the per-snapshot dict prologue. slot
// indexes into the slot-keyed dictSlotIDs (per-snapshot, reassigned every
// state); row indexes into the stable card-body table for the body splice.
func (e *directTokenEmitter) emitDictEntry(slot, row int32) {
	e.closeScalarOwner()
	if slot >= 0 && slot < int32(len(e.tables.dictSlotIDs)) {
		e.writeSingle(e.tables.dictSlotIDs[slot])
	} else if row >= 0 && row < int32(len(e.tables.dictEntryIDs)) {
		e.writeSingle(e.tables.dictEntryIDs[row])
	}
	if row >= 0 && row < e.tables.cardRowCount {
		e.writeSpan(e.tables.cardBodySpan(row))
	}
	e.writeSpan(e.tables.cardCloser)
}

func (e *directTokenEmitter) emitStackOpen() {
	e.closeScalarOwner()
	e.writeSingle(e.tables.stackOpenID)
}

func (e *directTokenEmitter) emitStackClose() {
	e.closeScalarOwner()
	e.writeSingle(e.tables.stackCloseID)
}

func (e *directTokenEmitter) finish() (int32, bool) {
	if e.overflow {
		endAbs := e.cursor + e.out.cursorBase
		for o := int32(0); o < e.out.maxOptions; o++ {
			if e.out.optionPos[o] >= endAbs {
				e.out.optionPos[o] = -1
				e.out.optionMask[o] = 0
			}
			for t := int32(0); t < e.out.maxTargets; t++ {
				idx := o*e.out.maxTargets + t
				if e.out.targetPos[idx] >= endAbs {
					e.out.targetPos[idx] = -1
					e.out.targetMask[idx] = 0
				}
			}
		}
		for k := int32(0); k < e.out.maxCardRefs; k++ {
			if e.out.cardRefPos[k] >= endAbs {
				e.out.cardRefPos[k] = -1
			}
		}
	}
	// Record what we dirtied so the next reset on this row can do a
	// partial clear. cardRefSeen is already populated in-place.
	e.dirty.optionWatermark = e.nextOption
	e.dirty.targetWatermark = e.maxTargetSlot
	return e.cursor, e.overflow
}
