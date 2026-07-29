package main

import (
	"fmt"
)

// Status-bit constants mirrored from render_plan.py / render_plan.go.
const (
	statusTappedKnown int32 = 0x2000
)

// Action-kind metadata flags. Bits set per kind id; consumers branch on
// flags rather than open-coded equality checks. kindFlagHasSource is set
// for every kind that participates in the source-row / ability suffix
// emission on OP_OPTION (everything except pass=0 and choice=6, matching
// the Python guard 'verb in ("pass", "choice", "unknown")'). kindFlagAbility
// is set only for activate_ability=3, which is the kind that consumes
// abilityIdx into a real span emit.
const (
	kindFlagHasSource uint8 = 1 << 0
	kindFlagAbility   uint8 = 1 << 1
)

// kindFlags is indexed by action-kind id. Sized at 16 to give room for
// future kinds without rebuilding; entries past actionKinds remain zero,
// which means "no source, no ability" — equivalent to treating unknown
// kinds as Python's "unknown".
var kindFlags = [16]uint8{
	0: 0,                                   // pass: no source, no ability
	1: kindFlagHasSource,                   // play_land
	2: kindFlagHasSource,                   // cast_spell
	3: kindFlagHasSource | kindFlagAbility, // activate_ability
	4: kindFlagHasSource,                   // attacker
	5: kindFlagHasSource,                   // blocker
	6: 0,                                   // choice: no source, no ability
}

func kindHasNoSource(kindID int32) bool {
	if kindID < 0 || int(kindID) >= len(kindFlags) {
		return true
	}
	return kindFlags[kindID]&kindFlagHasSource == 0
}

// MAX_CARD_REFS is mirrored from magic_ai/text_encoder/tokenizer.py.
const tokenAssemblerMaxCardRefs = 256

// Frag enum values mirrored from token_tables.py:Frag. Append-only.
const (
	fragBosState       int32 = 0
	fragCloseStateEos  int32 = 1
	fragCloseSelf      int32 = 2
	fragCloseOpp       int32 = 3
	fragCloseOption    int32 = 4
	fragOpenActions    int32 = 5
	fragCloseActions   int32 = 6
	fragOpenTarget     int32 = 7
	fragSpace          int32 = 8
	fragTargetFallback int32 = 9
	fragSelfMana       int32 = 10
	fragOppMana        int32 = 11
)

// Opcode arities mirrored from render_plan.py. Variable-length opcodes
// (OP_LITERAL_TOKENS) carry their length as the first payload word.
//
// Stored as a fixed-size array keyed by opcode rather than a map because
// the assembler hot loop indexes this on every opcode and a map lookup
// dominates the profile (mapaccess2_fast32 + memhash32 ~ half of CPU).
// Unknown opcodes return -1 from opcodeArityLookup.
var opcodeArityArr = [...]int8{
	opOpenState:     0,
	opCloseState:    0,
	opTurn:          2,
	opLife:          2,
	opMana:          3,
	opOpenPlayer:    1,
	opClosePlayer:   0,
	opOpenZone:      2,
	opCloseZone:     0,
	opPlaceCard:     4,
	opCounter:       2,
	opAttachedTo:    1,
	opOpenActions:   0,
	opCloseActions:  0,
	opOption:        5,
	opTarget:        3,
	opLiteralTokens: -1, // variable-length; handled separately in the walker
	opEndCard:       0,
	opOpenRawCard:   1,
	opCloseRawCard:  0,
	opOpenDict:      0,
	opCloseDict:     0,
	opDictEntry:     1,
	opPlaceCardRef:  4,
	opCount:         1,
	opStackOpen:     0,
	opStackClose:    0,
	opCommandOpen:   0,
	opCommandClose:  0,
}

// opcodeArityLookup returns (arity, true) for known opcodes and (0, false)
// otherwise. Mirrors a map lookup but avoids the hash-table cost.
func opcodeArityLookup(op int32) (int, bool) {
	if op <= 0 || int(op) >= len(opcodeArityArr) {
		return 0, false
	}
	a := opcodeArityArr[op]
	if a < 0 {
		return 0, false
	}
	return int(a), true
}

type zoneEntry struct {
	zone  int32
	owner int32
}

// tokenAssemblerOut bundles the per-batch-row output slices passed to the
// walker. Slices are views into the Python-allocated output tensors. -1
// sentinels mark absent positions; mask slices carry 0/1 indicators.
type tokenAssemblerOut struct {
	tokenIDs    []int32
	optionPos   []int32
	optionMask  []uint8
	targetPos   []int32 // length max_options*max_targets
	targetMask  []uint8 // length max_options*max_targets
	cardRefPos  []int32
	maxOptions  int32
	maxTargets  int32
	maxCardRefs int32
	// cursorBase shifts every recorded anchor position by a constant
	// offset before it is written. Packed mode passes the row's start offset
	// into the shared packed buffer so anchors land as absolute offsets.
	cursorBase int32
}

// assembleTokensFromPlan walks “plan“ (an int32 render-plan stream) and
// writes tokens + anchor positions into “out“. Mirrors the Python
// _assemble_one walker 1:1; out-of-bounds scalars (turn/life/ability)
// raise an error rather than falling back to a live encode.
//
// Returns the sequence length written and a truncation flag (true if the
// stream overran “maxTokens“ and was cut off).
func assembleTokensFromPlan(
	plan []int32,
	tables *tokenTables,
	out *tokenAssemblerOut,
	maxTokens int32,
) (tokensWritten int32, truncated bool, err error) {
	if tables == nil {
		return 0, false, fmt.Errorf("token tables not registered")
	}

	// Pre-fill anchor sentinels.
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

	// First-occurrence card-ref bitmap (matches Python's "k not in card_ref_positions").
	// 256-bit stack-resident bitset avoids a per-row [256]bool heap alloc.
	var cardRefSeen [tokenAssemblerMaxCardRefs / 64]uint64

	var (
		cursor          int32 // next write index in tokenIDs
		overflow        bool
		nextOption      int32      // index in option_positions for the next OP_OPTION
		curOptionIdx    int32 = -1 // index of the option whose target bucket is open
		curTargetCount  int32
		optionOpen      bool
		scalarOwnerOpen int32 = -1 // -1 / 0 / 1
	)
	// Stack-resident backing array for zoneStack avoids the heap alloc of
	// the first append. Plans nest at most a few zones deep; spillover
	// would just trigger a normal growslice.
	var zoneStackArr [8]zoneEntry
	zoneStack := zoneStackArr[:0]

	// Detect literal-tokens mode by scanning the opcode stream. Naive
	// "any token equals OP_LITERAL_TOKENS" misfires on payload ints.
	structured := true
	{
		i := 0
		for i < len(plan) {
			op := plan[i]
			if op == opLiteralTokens {
				structured = false
				break
			}
			arity, ok := opcodeArityLookup(op)
			if !ok {
				break
			}
			i += 1 + arity
		}
	}

	// Helpers ---------------------------------------------------------------

	writeSpan := func(span []int32) {
		if overflow || span == nil {
			return
		}
		n := int32(len(span))
		if cursor+n > maxTokens {
			// Write what fits, mark overflow, stop.
			room := maxTokens - cursor
			if room > 0 {
				copy(out.tokenIDs[cursor:cursor+room], span[:room])
				cursor += room
			}
			overflow = true
			return
		}
		copy(out.tokenIDs[cursor:cursor+n], span)
		cursor += n
	}

	writeSingle := func(id int32) int32 {
		if overflow {
			return -1
		}
		if cursor >= maxTokens {
			overflow = true
			return -1
		}
		pos := cursor
		out.tokenIDs[cursor] = id
		cursor++
		return pos
	}

	emitFragment := func(fragID int32) {
		writeSpan(tables.fragmentSpan(fragID))
	}

	// Returns true iff a card-ref token was emitted (matches Python).
	emitCardRef := func(uuidIdx int32) bool {
		if uuidIdx < 0 || uuidIdx >= tokenAssemblerMaxCardRefs || uuidIdx >= tables.cardRefCount {
			return false
		}
		refID := tables.cardRefIDs[uuidIdx]
		pos := writeSingle(refID)
		if pos < 0 {
			return false
		}
		mask := uint64(1) << (uint32(uuidIdx) & 63)
		word := uint32(uuidIdx) >> 6
		if cardRefSeen[word]&mask == 0 {
			cardRefSeen[word] |= mask
			out.cardRefPos[uuidIdx] = pos + out.cursorBase
		}
		return true
	}

	closeScalarOwner := func() {
		if scalarOwnerOpen < 0 {
			return
		}
		if scalarOwnerOpen == 0 {
			emitFragment(fragCloseSelf)
		} else {
			emitFragment(fragCloseOpp)
		}
		scalarOwnerOpen = -1
	}

	closeOption := func() {
		if optionOpen {
			emitFragment(fragCloseOption)
			optionOpen = false
		}
	}

	// Main walk -------------------------------------------------------------

	i := 0
	for i < len(plan) && !overflow {
		op := plan[i]
		arity, ok := opcodeArityLookup(op)
		if !ok {
			return 0, false, fmt.Errorf("unknown opcode %d at position %d", op, i)
		}

		if op == opLiteralTokens {
			closeScalarOwner()
			length := int(plan[i+1])
			start := i + 2
			end := start + length
			for j := start; j < end && !overflow; j++ {
				tid := plan[j]
				pos := writeSingle(tid)
				if pos < 0 {
					break
				}
				switch {
				case tid == tables.optionID:
					if nextOption < out.maxOptions {
						out.optionPos[nextOption] = pos + out.cursorBase
						out.optionMask[nextOption] = 1
						curOptionIdx = nextOption
						curTargetCount = 0
						nextOption++
					}
				case tid == tables.targetOpenID && curOptionIdx >= 0:
					if curTargetCount < out.maxTargets {
						idx := curOptionIdx*out.maxTargets + curTargetCount
						out.targetPos[idx] = pos + out.cursorBase
						out.targetMask[idx] = 1
						curTargetCount++
					}
				default:
					// card-ref ids: record first-occurrence position per K.
					for k := int32(0); k < tables.cardRefCount; k++ {
						if tables.cardRefIDs[k] == tid {
							mask := uint64(1) << (uint32(k) & 63)
							word := uint32(k) >> 6
							if cardRefSeen[word]&mask == 0 {
								cardRefSeen[word] |= mask
								out.cardRefPos[k] = pos + out.cursorBase
							}
							break
						}
					}
				}
			}
			i = end
			continue
		}

		if structured {
			if scalarOwnerOpen >= 0 {
				keepOpen := op == opMana && plan[i+1] == scalarOwnerOpen
				if !keepOpen {
					closeScalarOwner()
				}
			}

			switch op {
			case opOpenState:
				emitFragment(fragBosState)
				i++
				continue
			case opCloseState:
				closeOption()
				closeScalarOwner()
				emitFragment(fragCloseStateEos)
				i++
				continue
			case opTurn:
				turn := plan[i+1]
				stepID := plan[i+2]
				if stepID < 0 || stepID >= tables.stepCount {
					stepID = tables.stepCount - 1
				}
				span := tables.turnStepSpan(turn, stepID)
				if span == nil {
					return 0, false, fmt.Errorf(
						"OP_TURN out of bounds: turn=%d step=%d (range %d..%d)",
						turn, stepID, tables.turnMin, tables.turnMax,
					)
				}
				writeSpan(span)
				i += 1 + arity
				continue
			case opLife:
				closeScalarOwner()
				owner := plan[i+1]
				life := plan[i+2]
				span := tables.lifeOwnerSpan(life, owner)
				if span == nil {
					return 0, false, fmt.Errorf(
						"OP_LIFE out of bounds: life=%d owner=%d (range %d..%d)",
						life, owner, tables.lifeMin, tables.lifeMax,
					)
				}
				writeSpan(span)
				scalarOwnerOpen = owner
				i += 1 + arity
				continue
			case opMana:
				owner := plan[i+1]
				colorID := plan[i+2]
				amount := plan[i+3]
				if scalarOwnerOpen < 0 {
					if owner == 0 {
						emitFragment(fragSelfMana)
					} else {
						emitFragment(fragOppMana)
					}
					scalarOwnerOpen = owner
				}
				if colorID >= 0 && colorID < tables.manaColorCount && amount > 0 {
					glyph := tables.manaGlyphSpan(colorID)
					for r := int32(0); r < amount && !overflow; r++ {
						writeSpan(glyph)
					}
				}
				i += 1 + arity
				continue
			case opOpenZone:
				closeOption()
				zone := plan[i+1]
				owner := plan[i+2]
				zoneStack = append(zoneStack, zoneEntry{zone: zone, owner: owner})
				writeSpan(tables.zoneOpenSpan(zone, owner))
				i += 1 + arity
				continue
			case opCloseZone:
				closeOption()
				if len(zoneStack) > 0 {
					top := zoneStack[len(zoneStack)-1]
					zoneStack = zoneStack[:len(zoneStack)-1]
					writeSpan(tables.zoneCloseSpan(top.zone, top.owner))
				}
				i++
				continue
			case opOpenActions:
				emitFragment(fragOpenActions)
				i++
				continue
			case opCloseActions:
				closeOption()
				emitFragment(fragCloseActions)
				i++
				continue
			case opOption:
				closeOption()
				kindID := plan[i+1]
				sourceRow := plan[i+2]
				sourceUUIDIdx := plan[i+3]
				_ = plan[i+4] // mana_cost_id (unused)
				abilityIdx := plan[i+5]

				pos := writeSingle(tables.optionID)
				if pos >= 0 && nextOption < out.maxOptions {
					out.optionPos[nextOption] = pos + out.cursorBase
					out.optionMask[nextOption] = 1
					curOptionIdx = nextOption
					curTargetCount = 0
					nextOption++
				}
				optionOpen = true

				verbSpan := tables.actionVerbSpan(kindID)
				kindKnown := verbSpan != nil
				if !kindKnown {
					// Unknown kind: emit nothing for the verb. Python falls
					// back to a literal " unknown" encode, but the structured
					// Go emitter never produces unknown kinds in practice.
					// Mirror Python's "kindKnown && !pass/choice" guard so
					// the suffix is also skipped.
				} else {
					writeSpan(verbSpan)
				}
				if kindKnown && !kindHasNoSource(kindID) {
					if !emitCardRef(sourceUUIDIdx) {
						if sourceRow >= 0 && sourceRow < tables.cardRowCount {
							writeSpan(tables.cardNameSpan(sourceRow))
						}
					}
				}
				if abilityIdx >= 0 && kindID == 3 {
					span := tables.abilitySpan(abilityIdx)
					if span == nil {
						return 0, false, fmt.Errorf(
							"OP_OPTION out of bounds: ability=%d (range %d..%d)",
							abilityIdx, tables.abilityMin, tables.abilityMax,
						)
					}
					writeSpan(span)
				}
				i += 1 + arity
				continue
			case opTarget:
				targetRow := plan[i+1]
				targetUUIDIdx := plan[i+2]
				targetKind := plan[i+3]

				pos := writeSingle(tables.targetOpenID)
				if pos >= 0 && curOptionIdx >= 0 && curTargetCount < out.maxTargets {
					idx := curOptionIdx*out.maxTargets + curTargetCount
					out.targetPos[idx] = pos + out.cursorBase
					out.targetMask[idx] = 1
					curTargetCount++
				}
				// Player targets: targetRow encodes the owner index (0=self,
				// 1=opp); emit the corresponding singleton id. Permanent /
				// card-in-zone targets resolve to a card-ref or fall back to
				// the row's display-name span.
				switch targetKind {
				case 0: // renderTargetPlayer
					if targetRow == 0 {
						writeSingle(tables.selfID)
					} else {
						writeSingle(tables.oppID)
					}
				default:
					if !emitCardRef(targetUUIDIdx) {
						if targetRow >= 0 && targetRow < tables.cardRowCount {
							writeSpan(tables.cardNameSpan(targetRow))
						} else {
							emitFragment(fragTargetFallback)
						}
					}
				}
				writeSingle(tables.targetCloseID)
				i += 1 + arity
				continue
			case opCount:
				amount := plan[i+1]
				span := tables.countSpan(amount)
				if span == nil {
					return 0, false, fmt.Errorf(
						"OP_COUNT out of bounds: amount=%d (range %d..%d)",
						amount, tables.countMin, tables.countMax,
					)
				}
				writeSpan(span)
				i += 1 + arity
				continue
			case opStackOpen:
				writeSingle(tables.stackOpenID)
				i++
				continue
			case opStackClose:
				writeSingle(tables.stackCloseID)
				i++
				continue
			case opCommandOpen:
				writeSingle(tables.commandOpenID)
				i++
				continue
			case opCommandClose:
				writeSingle(tables.commandCloseID)
				i++
				continue
			}
		}

		// Both structured and literal-tokens mode reach here for these:

		switch op {
		case opPlaceCard:
			_ = plan[i+1] // slot_idx (unused)
			row := plan[i+2]
			status := plan[i+3]
			uuidIdx := plan[i+4]
			emitCardRef(uuidIdx)
			if row >= 0 && row < tables.cardRowCount {
				writeSpan(tables.cardBodySpan(row))
			}
			if status&statusTappedKnown != 0 {
				if status&0x0001 != 0 {
					writeSpan(tables.statusTapped)
				} else {
					writeSpan(tables.statusUntapped)
				}
			} else if structured && status&0x0001 != 0 {
				writeSpan(tables.statusTapped)
			}
			if structured {
				writeSpan(tables.cardCloser)
			}
			i += 1 + arity
			continue
		case opEndCard:
			writeSpan(tables.cardCloser)
			i++
			continue
		case opOpenRawCard:
			uuidIdx := plan[i+1]
			emitCardRef(uuidIdx)
			i += 1 + arity
			continue
		case opOpenDict:
			writeSingle(tables.dictOpenID)
			i++
			continue
		case opCloseDict:
			writeSingle(tables.dictCloseID)
			i++
			continue
		case opDictEntry:
			row := plan[i+1]
			if row >= 0 && row < int32(len(tables.dictEntryIDs)) {
				writeSingle(tables.dictEntryIDs[row])
			}
			if row >= 0 && row < tables.cardRowCount {
				writeSpan(tables.cardBodySpan(row))
			}
			writeSpan(tables.cardCloser)
			i += 1 + arity
			continue
		case opPlaceCardRef:
			_ = plan[i+1] // slot_idx (unused)
			row := plan[i+2]
			status := plan[i+3]
			uuidIdx := plan[i+4]
			emitCardRef(uuidIdx)
			writeSingle(tables.cardOpenID)
			if row >= 0 && row < int32(len(tables.dictEntryIDs)) {
				writeSingle(tables.dictEntryIDs[row])
			}
			if status&statusTappedKnown != 0 {
				if status&0x0001 != 0 {
					writeSpan(tables.statusTapped)
				} else {
					writeSpan(tables.statusUntapped)
				}
			} else if structured && status&0x0001 != 0 {
				writeSpan(tables.statusTapped)
			}
			writeSpan(tables.cardCloser)
			i += 1 + arity
			continue
		}

		// Bookkeeping-only opcodes — skip over header + payload.
		switch op {
		case opOpenState, opCloseState, opOpenZone, opCloseZone,
			opOpenActions, opCloseActions, opOpenPlayer, opClosePlayer,
			opCounter, opAttachedTo, opOption, opTarget, opTurn, opLife,
			opMana, opCloseRawCard, opOpenDict, opCloseDict, opDictEntry,
			opPlaceCardRef, opCount, opStackOpen, opStackClose,
			opCommandOpen, opCommandClose:
			i += 1 + arity
			continue
		}

		return 0, false, fmt.Errorf("unhandled opcode %d at position %d", op, i)
	}

	// Truncation: mirrors the Python "preserve K↔K invariant" path. Option
	// positions past max_tokens become -1 sentinels; option_mask stays 1
	// (the option exists) but option_position is unreachable. Same for
	// targets and card-refs.
	if overflow {
		// Compare against the absolute end-of-row offset so truncation tests
		// packed absolute anchors against the row-local token budget.
		endAbs := cursor + out.cursorBase
		for o := int32(0); o < out.maxOptions; o++ {
			if out.optionPos[o] >= endAbs {
				out.optionPos[o] = -1
				// Note: Python keeps option_mask=False for truncated slots
				// because the assembler clears it when option_pos == -1.
				// Match that.
				out.optionMask[o] = 0
			}
			for t := int32(0); t < out.maxTargets; t++ {
				idx := o*out.maxTargets + t
				if out.targetPos[idx] >= endAbs {
					out.targetPos[idx] = -1
					out.targetMask[idx] = 0
				}
			}
		}
		for k := int32(0); k < out.maxCardRefs; k++ {
			if out.cardRefPos[k] >= endAbs {
				out.cardRefPos[k] = -1
			}
		}
	}

	return cursor, overflow, nil
}
