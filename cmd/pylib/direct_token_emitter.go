package main

import "fmt"

type directTokenEmitter struct {
	tables          *tokenTables
	out             *tokenAssemblerOut
	maxTokens       int32
	cursor          int32
	overflow        bool
	cardRefSeen     [tokenAssemblerMaxCardRefs]bool
	nextOption      int32
	curOptionIdx    int32
	curTargetCount  int32
	optionOpen      bool
	scalarOwnerOpen int32
}

func newDirectTokenEmitter(tables *tokenTables, out *tokenAssemblerOut, maxTokens int32) *directTokenEmitter {
	e := &directTokenEmitter{}
	e.reset(tables, out, maxTokens)
	return e
}

// reset re-binds an emitter to a new output row and clears the per-row
// state. Lets callers keep a single emitter on the heap (e.g. on
// encodeScratch) and reuse it across batch rows; the [256]bool
// cardRefSeen field stays in place rather than getting re-zeroed every
// row, which is the dominant per-row alloc/cost in the direct path.
func (e *directTokenEmitter) reset(tables *tokenTables, out *tokenAssemblerOut, maxTokens int32) {
	for i := range out.optionPos {
		out.optionPos[i] = -1
	}
	for i := range out.optionMask {
		out.optionMask[i] = 0
	}
	for i := range out.targetPos {
		out.targetPos[i] = -1
	}
	for i := range out.targetMask {
		out.targetMask[i] = 0
	}
	for i := range out.cardRefPos {
		out.cardRefPos[i] = -1
	}
	e.tables = tables
	e.out = out
	e.maxTokens = maxTokens
	e.cursor = 0
	e.overflow = false
	for i := range e.cardRefSeen {
		e.cardRefSeen[i] = false
	}
	e.nextOption = 0
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
	if !e.cardRefSeen[uuidIdx] {
		e.cardRefSeen[uuidIdx] = true
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
	if kindKnown {
		e.writeSpan(verbSpan)
	}
	if kindKnown && !kindHasNoSource(kindID) {
		if !e.emitCardRef(sourceUUIDIdx) {
			if sourceRow >= 0 && sourceRow < e.tables.cardRowCount {
				e.writeSpan(e.tables.cardNameSpan(sourceRow))
			}
		}
	}
	if abilityIdx >= 0 && kindID == 3 {
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

func (e *directTokenEmitter) emitPlaceCardRef(row, status, uuidIdx int32) {
	e.closeScalarOwner()
	e.emitCardRef(uuidIdx)
	e.writeSingle(e.tables.cardOpenID)
	if row >= 0 && row < int32(len(e.tables.dictEntryIDs)) {
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

func (e *directTokenEmitter) emitOpenDict() {
	e.closeScalarOwner()
	e.writeSingle(e.tables.dictOpenID)
}

func (e *directTokenEmitter) emitCloseDict() {
	e.closeScalarOwner()
	e.writeSingle(e.tables.dictCloseID)
}

func (e *directTokenEmitter) emitDictEntry(row int32) {
	e.closeScalarOwner()
	if row >= 0 && row < int32(len(e.tables.dictEntryIDs)) {
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
	return e.cursor, e.overflow
}
