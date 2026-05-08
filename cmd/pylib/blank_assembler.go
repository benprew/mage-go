package main

import "fmt"

// blankCollector records EMIT_BLANK / EMIT_BLANK_LEGAL output for one batch
// row. Slices are views into caller-allocated tensors with batch-wide max
// dimensions K (max blanks per row) and Vmax (max legal ids per blank).
//
// Layout (all row-local — caller is responsible for slicing into the [B,...]
// tensors before invoking the walker):
//
//	positions   [K]      int32  — token-stream index of the kind token
//	kind        [K]      int32  — kind token id (e.g. choose_target_id)
//	group       [K]      int32  — group_id within the snapshot
//	groupKind   [K]      int32  — blankGroup{PerBlank,CrossBlank,Constrained}
//	optionIndex [K]      int32  — engine option index, -1 for non-option blanks
//	legalIDs    [K*Vmax] int32  — flat row-major legal-id buffer
//	legalMask   [K*Vmax] uint8  — 1 for valid entries, 0 for padding
type blankCollector struct {
	positions  []int32
	kind       []int32
	group      []int32
	groupKind  []int32
	optionIdx  []int32
	legalIDs   []int32
	legalMask  []uint8
	count      *int32
	legalCount []int32
	overflow   *int32

	maxBlanks      int32
	maxLegalPerBlk int32

	// Live state during a walk.
	blankCount  int32 // number of EMIT_BLANK seen so far this row
	curLegalK   int32 // index of the in-progress blank (== blankCount-1 while filling)
	curLegalN   int32 // number of legal ids written for the current blank
	curLegalCap int32 // declared legal_count for the current blank
}

// reset clears only live-count metadata and resets the live state. Stale
// padded tail slots are ignored by the exported count arrays.
func (c *blankCollector) reset(maxBlanks, maxLegalPerBlk int32) {
	c.maxBlanks = maxBlanks
	c.maxLegalPerBlk = maxLegalPerBlk
	c.blankCount = 0
	c.curLegalK = -1
	c.curLegalN = 0
	c.curLegalCap = 0
	if c.count != nil {
		*c.count = 0
	}
}

// recordBlank writes the metadata for a new EMIT_BLANK and primes the legal
// list. cursor is the absolute (or row-local — caller's choice) token-stream
// position where the kind token was just written.
func (c *blankCollector) recordBlank(cursor, kindID, groupID, groupKind, optionIndex, legalCount int32) error {
	if c.blankCount >= c.maxBlanks {
		// Silently drop excess blanks (matches the existing options/targets
		// truncation policy). The caller tracks this via the row's tokens-
		// overflow flag; an explicit blanks-overflow flag can be added when
		// Step 3 wires the Python side through.
		c.curLegalK = -1
		c.curLegalCap = 0
		c.curLegalN = 0
		if c.overflow != nil {
			*c.overflow = *c.overflow + 1
		}
		return nil
	}
	if legalCount < 0 {
		return fmt.Errorf("EMIT_BLANK legal_count=%d must be non-negative", legalCount)
	}
	idx := c.blankCount
	c.positions[idx] = cursor
	c.kind[idx] = kindID
	c.group[idx] = groupID
	c.groupKind[idx] = groupKind
	if len(c.legalCount) > int(idx) {
		c.legalCount[idx] = 0
	}
	if len(c.optionIdx) > int(idx) {
		c.optionIdx[idx] = optionIndex
	}
	c.curLegalK = idx
	c.curLegalCap = legalCount
	c.curLegalN = 0
	c.blankCount++
	if c.count != nil {
		*c.count = c.blankCount
	}
	return nil
}

// recordLegal appends one legal token id to the in-progress blank.
func (c *blankCollector) recordLegal(tokenID int32) error {
	if c.curLegalK < 0 {
		// Either we exceeded maxBlanks and are dropping, or no EMIT_BLANK
		// has primed the buffer. The latter is a wire-format bug; surface it.
		if c.blankCount >= c.maxBlanks {
			return nil
		}
		return fmt.Errorf("EMIT_BLANK_LEGAL with no preceding EMIT_BLANK")
	}
	if c.curLegalN >= c.curLegalCap {
		return fmt.Errorf("EMIT_BLANK_LEGAL count exceeds declared legal_count=%d", c.curLegalCap)
	}
	if c.curLegalN < c.maxLegalPerBlk {
		base := c.curLegalK * c.maxLegalPerBlk
		c.legalIDs[base+c.curLegalN] = tokenID
		c.legalMask[base+c.curLegalN] = 1
	}
	c.curLegalN++
	if len(c.legalCount) > int(c.curLegalK) {
		c.legalCount[c.curLegalK] = c.curLegalN
	}
	return nil
}

// finalize sanity-checks that every primed legal list was filled.
func (c *blankCollector) finalize() error {
	if c.curLegalK >= 0 && c.curLegalN != c.curLegalCap {
		return fmt.Errorf(
			"EMIT_BLANK at index %d declared legal_count=%d but received %d",
			c.curLegalK, c.curLegalCap, c.curLegalN,
		)
	}
	return nil
}

// walkBlankPlan walks a render plan and processes the EMIT_BLANK /
// EMIT_BLANK_LEGAL opcodes. All other opcodes are skipped over (their token
// emission is handled by assembleTokensFromPlan / directTokenEmitter — this
// pass exists only to populate the blank-anchor outputs). Returns an error
// for malformed legal-list sequences.
//
// cursorAtBlank is the absolute or row-local cursor position to record for
// each EMIT_BLANK opcode. Real callers pair this with a token-emission walk
// and pass the cursor that walk has reached at the moment EMIT_BLANK fires.
// For unit testing, the caller can pass a single fixed cursor or zero.
func walkBlankPlan(plan []int32, collector *blankCollector, cursorFn func() int32) error {
	if collector == nil {
		return nil
	}
	i := 0
	for i < len(plan) {
		op := plan[i]
		arity, ok := opcodeArityLookup(op)
		if !ok {
			if op == opLiteralTokens {
				if i+1 >= len(plan) {
					return fmt.Errorf("opLiteralTokens missing length at %d", i)
				}
				length := int(plan[i+1])
				i += 2 + length
				continue
			}
			return fmt.Errorf("unknown opcode %d at %d", op, i)
		}
		switch op {
		case opEmitBlank:
			kindID := plan[i+1]
			groupID := plan[i+2]
			groupKind := plan[i+3]
			legalCount := plan[i+4]
			cursor := int32(0)
			if cursorFn != nil {
				cursor = cursorFn()
			}
			if err := collector.recordBlank(cursor, kindID, groupID, groupKind, -1, legalCount); err != nil {
				return err
			}
		case opEmitBlankLegal:
			if err := collector.recordLegal(plan[i+1]); err != nil {
				return err
			}
		}
		i += 1 + arity
	}
	return collector.finalize()
}
