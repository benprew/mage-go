package main

import (
	"github.com/google/uuid"
)

// DecisionType / AnchorKind enums mirror the Python definitions in
// magic_ai/text_encoder/decision_spec.py. Stable across the wire; append-only.
type decisionType int32

const (
	decTypePriority         decisionType = 0
	decTypeDeclareAttackers decisionType = 1
	decTypeDeclareBlockers  decisionType = 2
	decTypeChooseTargets    decisionType = 3
	decTypeMay              decisionType = 4
	decTypeChooseMode       decisionType = 5
	decTypeChooseX          decisionType = 6
	decTypeNone             decisionType = -1
)

type anchorKind int32

const (
	anchorLegalAttacker anchorKind = 0
	anchorLegalBlocker  anchorKind = 1
	anchorLegalTarget   anchorKind = 2
	anchorLegalAction   anchorKind = 3
	anchorDefender      anchorKind = 4
)

// specTokenIDs holds the spec-section structural-tag token ids that Python
// supplies via the extension to MageRegisterTokenTables (see specTagIDs in
// the abi.h appendix added by step 8). Field order mirrors the abi.h block
// labelled "Decision-spec tag tokens" — keep them in lockstep.
type specTokenIDs struct {
	specOpen      int32
	specClose     int32
	decisionType  int32
	legalAttacker int32
	legalBlocker  int32
	legalTarget   int32
	legalAction   int32
	forAction     int32
	maxValueOpen  int32
	maxValueClose int32
	playerRef0    int32
	playerRef1    int32
	// dtName[k] is the token id for `<dt-{name}>` for decisionType k
	// (0=priority, 1=declare_attackers, ..., 6=choose_x).
	dtName [7]int32
	// stackRef[k] is the token id for `<stack-ref:k>`. May be empty if
	// stack refs aren't yet in the tokenizer; emitter falls back to
	// playerRef0 when out of range and the test path sets length 16.
	stackRef []int32
}

// pointerAnchor mirrors the Python PointerAnchor. tokenPosition is in spec-
// local coordinates here; the caller shifts by stateTokens length before
// committing to the combined-stream output buffer.
type pointerAnchor struct {
	kind          anchorKind
	tokenPosition int32
	subjectIndex  int32
	handle        int32
}

// specEmitterOut is the per-row receiver for spec emission. The caller
// pre-allocates capacity (spec_tokens cap, anchors cap, bitmap cap) and
// resets between rows. After emission, spec_tokens[:tokensLen] is the live
// region; anchors[:anchorsLen] are the live anchors. nBlockers / nAttackers
// shape legalEdgeBitmap (row-major bool, 0/1).
type specEmitterOut struct {
	tokens               []int32
	tokensLen            int32
	anchors              []pointerAnchor
	anchorsLen           int32
	maxValueDigits       []int32 // BPE digit-id table provided by Python; see below
	maxValueDigitOffsets []int32
	maxValueDigitMax     int32
	// Output side-tensor (DECLARE_BLOCKERS only).
	legalEdgeBitmap []byte
	nBlockers       int32
	nAttackers      int32
	// Decision type emitted for this row, or decTypeNone if no pending.
	decisionType decisionType
}

func (o *specEmitterOut) reset() {
	o.tokensLen = 0
	o.anchorsLen = 0
	o.nBlockers = 0
	o.nAttackers = 0
	o.decisionType = decTypeNone
	if len(o.legalEdgeBitmap) > 0 {
		clear(o.legalEdgeBitmap)
	}
}

func (o *specEmitterOut) emit(tok int32) {
	if int(o.tokensLen) >= len(o.tokens) {
		return
	}
	o.tokens[o.tokensLen] = tok
	o.tokensLen++
}

func (o *specEmitterOut) anchor(kind anchorKind, subjectIndex, handle int32) {
	if int(o.anchorsLen) >= len(o.anchors) {
		return
	}
	o.anchors[o.anchorsLen] = pointerAnchor{
		kind:          kind,
		tokenPosition: o.tokensLen,
		subjectIndex:  subjectIndex,
		handle:        handle,
	}
	o.anchorsLen++
}

// pendingKindToDecisionType mirrors render_spec.py's _PENDING_KIND_TO_DECISION_TYPE.
func pendingKindToDecisionType(kind string) (decisionType, bool) {
	switch kind {
	case "priority":
		return decTypePriority, true
	case "attackers":
		return decTypeDeclareAttackers, true
	case "blockers":
		return decTypeDeclareBlockers, true
	case "permanent", "cards_from_hand", "card_from_library":
		return decTypeChooseTargets, true
	case "may":
		return decTypeMay, true
	case "mode":
		return decTypeChooseMode, true
	case "number":
		return decTypeChooseX, true
	}
	return decTypeNone, false
}

// emitDecisionSpec writes one row's worth of spec tokens + anchors + side-
// tensors into out. Mirrors magic_ai/text_encoder/render_spec.py::DecisionSpecRenderer.render
// for the v1 anchor layout. Returns nil on success; an error means the
// pending kind is unknown or unsupported (callers should treat it like a
// no-spec row, leaving out.decisionType == decTypeNone).
func emitDecisionSpec(pending *apiPending, ids *specTokenIDs, out *specEmitterOut) error {
	out.reset()
	if pending == nil {
		return nil
	}
	dt, ok := pendingKindToDecisionType(pending.Kind)
	if !ok {
		// mana_color, mulligan, unknown — deferred per the plan.
		return nil
	}
	out.decisionType = dt

	out.emit(ids.specOpen)
	out.emit(ids.decisionType)
	out.emit(ids.dtName[dt])

	options := pending.Options

	switch dt {
	case decTypePriority:
		for optIdx := range options {
			out.anchor(anchorLegalAction, int32(optIdx), int32(optIdx))
			out.emit(ids.legalAction)
		}

	case decTypeDeclareAttackers:
		for optIdx := range options {
			out.anchor(anchorLegalAttacker, int32(optIdx), int32(optIdx))
			out.emit(ids.legalAttacker)
		}
		for playerIdx := range int32(2) {
			out.anchor(anchorDefender, playerIdx, playerIdx)
			if playerIdx == 0 {
				out.emit(ids.playerRef0)
			} else {
				out.emit(ids.playerRef1)
			}
		}

	case decTypeDeclareBlockers:
		// Build attacker order = first-seen union across all blocker valid_targets.
		var attackerOrder []uuid.UUID
		attackerIndex := map[uuid.UUID]int32{}
		for _, opt := range options {
			for _, t := range opt.ValidTargets {
				if t.IDUUID == uuid.Nil {
					continue
				}
				if _, exists := attackerIndex[t.IDUUID]; !exists {
					attackerIndex[t.IDUUID] = int32(len(attackerOrder))
					attackerOrder = append(attackerOrder, t.IDUUID)
				}
			}
		}
		// Emit blocker anchors.
		for optIdx := range options {
			out.anchor(anchorLegalBlocker, int32(optIdx), int32(optIdx))
			out.emit(ids.legalBlocker)
		}
		// Emit attacker anchors (re-emitted as spec anchors per render_spec.py
		// v1 trade-off comment).
		for atkIdx := range attackerOrder {
			out.anchor(anchorLegalAttacker, int32(atkIdx), int32(atkIdx))
			out.emit(ids.legalAttacker)
		}
		// Legal-edge bitmap [n_blockers, n_attackers].
		nB := int32(len(options))
		nA := int32(len(attackerOrder))
		out.nBlockers = nB
		out.nAttackers = nA
		need := int(nB) * int(nA)
		if cap(out.legalEdgeBitmap) < need {
			out.legalEdgeBitmap = make([]byte, need)
		} else {
			out.legalEdgeBitmap = out.legalEdgeBitmap[:need]
			clear(out.legalEdgeBitmap)
		}
		for blkIdx, opt := range options {
			for _, t := range opt.ValidTargets {
				if t.IDUUID == uuid.Nil {
					continue
				}
				if atkIdx, ok := attackerIndex[t.IDUUID]; ok {
					out.legalEdgeBitmap[int32(blkIdx)*nA+atkIdx] = 1
				}
			}
		}

	case decTypeChooseTargets:
		for optIdx := range options {
			out.anchor(anchorLegalTarget, int32(optIdx), int32(optIdx))
			out.emit(ids.legalTarget)
		}

	case decTypeMay:
		// Fixed grammar — no anchors, no body tokens.

	case decTypeChooseMode:
		maxValue := int32(len(options))
		out.emit(ids.maxValueOpen)
		emitDigits(out, ids, maxValue)
		out.emit(ids.maxValueClose)

	case decTypeChooseX:
		// CHOOSE_X: prefer pending.Amount when set; otherwise use
		// max(0, len(options)-1). Mirrors render_spec.py CHOOSE_X branch.
		var maxValue int32
		if pending.Amount > 0 {
			maxValue = int32(pending.Amount)
		} else if n := len(options); n > 0 {
			maxValue = int32(n - 1)
		}
		out.emit(ids.maxValueOpen)
		emitDigits(out, ids, maxValue)
		out.emit(ids.maxValueClose)
	}

	out.emit(ids.specClose)
	return nil
}

// emitDigits writes the BPE token id sequence for the integer “value“ by
// indexing into the precomputed digit-id table that Python uploads via
// MageRegisterTokenTables (see MaxValueDigitTokenIDsBuf). Values out of
// range fall back to a single digit-zero emission to preserve grammar
// well-formedness; production code must size the table to cover the full
// reachable max-value range (X cap, mode count cap).
func emitDigits(out *specEmitterOut, ids *specTokenIDs, value int32) {
	if value < 0 {
		value = 0
	}
	if out.maxValueDigitMax > 0 && value <= out.maxValueDigitMax &&
		len(out.maxValueDigitOffsets) > int(value)+1 {
		start := out.maxValueDigitOffsets[value]
		end := out.maxValueDigitOffsets[value+1]
		for i := start; i < end; i++ {
			out.emit(out.maxValueDigits[i])
		}
		return
	}
	// Fallback: emit ASCII-decimal digit tokens directly — only safe when
	// the tokenizer happens to register single-digit ids in the same table.
	// In production the offsets table always covers the reachable range,
	// so this is unreachable on the live path.
	if value == 0 {
		// nothing; emit a single zero placeholder if the table isn't wired
		// yet so tests that exercise the grammar shell don't crash
		out.emit(0)
		return
	}
	// Decompose to base-10 digits, MSB first.
	var digits [11]int32
	n := 0
	for v := value; v > 0; v /= 10 {
		digits[n] = v % 10
		n++
	}
	for i := n - 1; i >= 0; i-- {
		out.emit(digits[i])
	}
}
