package main

// Grammar vocab token ids — must stay in lockstep with magic_ai/text_encoder/grammar.py::GrammarVocab.
const (
	gPad                 int32 = 0
	gEnd                 int32 = 1
	gDeclareAttackersOpn int32 = 2
	gAttack              int32 = 3
	gDefender            int32 = 4
	gDeclareBlockersOpn  int32 = 5
	gBlock               int32 = 6
	gAttacker            int32 = 7
	gPriorityOpen        int32 = 8
	gChooseTargetsOpen   int32 = 9
	gChooseModeOpen      int32 = 10
	gChooseXOpen         int32 = 11
	gMayOpen             int32 = 12
	gYes                 int32 = 13
	gNo                  int32 = 14
	gNone                int32 = 15
	gDigit0              int32 = 16
	gDigit9              int32 = 25
	grammarVocabSize     int32 = 26
)

// decisionMaskInput packages the per-row state the mask machine needs.
// In the MageDecisionMaskNext callback, this is reconstructed from the
// batch handle's captured per-decision spec data.
type decisionMaskInput struct {
	decType decisionType
	// Spec-derived shapes.
	nLegalAttackers int32
	nLegalBlockers  int32
	nLegalTargets   int32
	nLegalActions   int32
	nDefenders      int32
	maxValue        int32
	// Side-tensor: legal-edge bitmap [nLegalBlockers, nLegalAttackers] for
	// DECLARE_BLOCKERS. Row-major, 1 = legal edge. Empty for other types.
	legalEdgeBitmap []byte
	// Prefix.
	prefixTokens   []int32
	prefixPointers []int32
	prefixLen      int32
}

// decisionMaskOutput is the per-row mask. vocabMask is exactly grammarVocabSize
// entries; pointerMask has caller-determined width (n_anchors_max), and entries
// past the live anchor count for the current step are left zero.
type decisionMaskOutput struct {
	vocabMask   []byte // [grammarVocabSize]
	pointerMask []byte // [n_anchors_max]
}

func clearMask(out *decisionMaskOutput) {
	for i := range out.vocabMask {
		out.vocabMask[i] = 0
	}
	for i := range out.pointerMask {
		out.pointerMask[i] = 0
	}
}

func setVocab(out *decisionMaskOutput, ids ...int32) {
	for _, id := range ids {
		if id >= 0 && int(id) < len(out.vocabMask) {
			out.vocabMask[id] = 1
		}
	}
}

func setPointersAll(out *decisionMaskOutput, n int32) {
	limit := min(int(n), len(out.pointerMask))
	for i := 0; i < limit; i++ {
		out.pointerMask[i] = 1
	}
}

// nextMask is the Go mirror of grammar.py::next_mask. “out“ must be pre-
// cleared; this function only sets bits, never resets them across the call.
// Returns an error on prefixes inconsistent with the grammar (the decoder
// should never produce these on the live path).
func nextMask(in *decisionMaskInput, out *decisionMaskOutput) error {
	clearMask(out)
	if len(in.prefixTokens) < int(in.prefixLen) || len(in.prefixPointers) < int(in.prefixLen) {
		return errMask("prefix slice shorter than prefixLen")
	}
	switch in.decType {
	case decTypePriority:
		return maskPriority(in, out)
	case decTypeDeclareAttackers:
		return maskDeclareAttackers(in, out)
	case decTypeDeclareBlockers:
		return maskDeclareBlockers(in, out)
	case decTypeChooseTargets:
		return maskChooseTargets(in, out)
	case decTypeMay:
		return maskMay(in, out)
	case decTypeChooseMode:
		return maskChooseInt(in, out, gChooseModeOpen)
	case decTypeChooseX:
		return maskChooseInt(in, out, gChooseXOpen)
	}
	return errMask("unknown decision type")
}

type maskErr string

func (e maskErr) Error() string { return string(e) }

func errMask(msg string) error { return maskErr(msg) }

func maskPriority(in *decisionMaskInput, out *decisionMaskOutput) error {
	switch in.prefixLen {
	case 0:
		setVocab(out, gPriorityOpen)
		return nil
	case 1:
		if in.prefixTokens[0] != gPriorityOpen {
			return errMask("PRIORITY: prefix must start with PRIORITY_OPEN")
		}
		setPointersAll(out, in.nLegalActions)
		return nil
	case 2:
		setVocab(out, gEnd)
		return nil
	}
	return errMask("PRIORITY: prefix too long")
}

func maskDeclareAttackers(in *decisionMaskInput, out *decisionMaskOutput) error {
	if in.prefixLen == 0 {
		setVocab(out, gDeclareAttackersOpn)
		return nil
	}
	if in.prefixTokens[0] != gDeclareAttackersOpn {
		return errMask("DECLARE_ATTACKERS: prefix must start with OPEN")
	}
	body := in.prefixTokens[1:in.prefixLen]
	bodyPtrs := in.prefixPointers[1:in.prefixLen]
	chosen := map[int32]struct{}{}
	i := int32(0)
	for i < int32(len(body)) {
		if body[i] != gAttack {
			return errMask("DECLARE_ATTACKERS: expected ATTACK in body")
		}
		if i+1 >= int32(len(body)) {
			// Need attacker pointer.
			limit := min(int(in.nLegalAttackers), len(out.pointerMask))
			for k := 0; k < limit; k++ {
				if _, taken := chosen[int32(k)]; !taken {
					out.pointerMask[k] = 1
				}
			}
			return nil
		}
		atkPtr := bodyPtrs[i+1]
		if _, taken := chosen[atkPtr]; taken {
			return errMask("DECLARE_ATTACKERS: attacker chosen twice")
		}
		chosen[atkPtr] = struct{}{}
		if i+2 >= int32(len(body)) {
			setVocab(out, gDefender)
			return nil
		}
		if body[i+2] != gDefender {
			return errMask("DECLARE_ATTACKERS: expected DEFENDER token")
		}
		if i+3 >= int32(len(body)) {
			setPointersAll(out, in.nDefenders)
			return nil
		}
		i += 4
	}
	if int32(len(chosen)) >= in.nLegalAttackers {
		setVocab(out, gEnd)
		return nil
	}
	setVocab(out, gAttack, gEnd)
	return nil
}

func maskDeclareBlockers(in *decisionMaskInput, out *decisionMaskOutput) error {
	if in.prefixLen == 0 {
		setVocab(out, gDeclareBlockersOpn)
		return nil
	}
	if in.prefixTokens[0] != gDeclareBlockersOpn {
		return errMask("DECLARE_BLOCKERS: prefix must start with OPEN")
	}
	body := in.prefixTokens[1:in.prefixLen]
	bodyPtrs := in.prefixPointers[1:in.prefixLen]
	chosenBlockers := map[int32]struct{}{}
	i := int32(0)
	for i < int32(len(body)) {
		if body[i] != gBlock {
			return errMask("DECLARE_BLOCKERS: expected BLOCK in body")
		}
		if i+1 >= int32(len(body)) {
			limit := min(int(in.nLegalBlockers), len(out.pointerMask))
			for k := 0; k < limit; k++ {
				if _, taken := chosenBlockers[int32(k)]; !taken {
					out.pointerMask[k] = 1
				}
			}
			return nil
		}
		blkPtr := bodyPtrs[i+1]
		if _, taken := chosenBlockers[blkPtr]; taken {
			return errMask("DECLARE_BLOCKERS: blocker chosen twice")
		}
		chosenBlockers[blkPtr] = struct{}{}
		if i+2 >= int32(len(body)) {
			setVocab(out, gAttacker)
			return nil
		}
		if body[i+2] != gAttacker {
			return errMask("DECLARE_BLOCKERS: expected ATTACKER token")
		}
		if i+3 >= int32(len(body)) {
			// Edge mask row for this blocker.
			if len(in.legalEdgeBitmap) == 0 {
				setPointersAll(out, in.nLegalAttackers)
				return nil
			}
			rowStart := blkPtr * in.nLegalAttackers
			limit := min(int(in.nLegalAttackers), len(out.pointerMask))
			for k := 0; k < limit; k++ {
				out.pointerMask[k] = in.legalEdgeBitmap[rowStart+int32(k)]
			}
			return nil
		}
		i += 4
	}
	if int32(len(chosenBlockers)) >= in.nLegalBlockers {
		setVocab(out, gEnd)
		return nil
	}
	setVocab(out, gBlock, gEnd)
	return nil
}

func maskChooseTargets(in *decisionMaskInput, out *decisionMaskOutput) error {
	switch in.prefixLen {
	case 0:
		setVocab(out, gChooseTargetsOpen)
		return nil
	case 1:
		if in.prefixTokens[0] != gChooseTargetsOpen {
			return errMask("CHOOSE_TARGETS: must start with OPEN")
		}
		setPointersAll(out, in.nLegalTargets)
		return nil
	case 2:
		setVocab(out, gEnd)
		return nil
	}
	return errMask("CHOOSE_TARGETS: prefix too long")
}

func maskMay(in *decisionMaskInput, out *decisionMaskOutput) error {
	switch in.prefixLen {
	case 0:
		setVocab(out, gMayOpen)
		return nil
	case 1:
		if in.prefixTokens[0] != gMayOpen {
			return errMask("MAY: must start with MAY_OPEN")
		}
		setVocab(out, gYes, gNo)
		return nil
	case 2:
		t := in.prefixTokens[1]
		if t != gYes && t != gNo {
			return errMask("MAY: second token must be YES or NO")
		}
		setVocab(out, gEnd)
		return nil
	}
	return errMask("MAY: prefix too long")
}

func maskChooseInt(in *decisionMaskInput, out *decisionMaskOutput, openTok int32) error {
	if in.maxValue < 0 {
		return errMask("CHOOSE_INT: max_value must be >= 0")
	}
	if in.prefixLen == 0 {
		setVocab(out, openTok)
		return nil
	}
	if in.prefixTokens[0] != openTok {
		return errMask("CHOOSE_INT: prefix must start with the right open token")
	}
	digits := in.prefixTokens[1:in.prefixLen]
	current := int32(0)
	for _, tok := range digits {
		if tok < gDigit0 || tok > gDigit9 {
			return errMask("CHOOSE_INT: expected digit token")
		}
		current = current*10 + (tok - gDigit0)
		if current > in.maxValue {
			return errMask("CHOOSE_INT: prefix integer exceeds max")
		}
	}
	base := current * 10
	if base <= in.maxValue {
		upper := int32(9)
		if in.maxValue-base < 9 {
			upper = in.maxValue - base
		}
		for d := int32(0); d <= upper; d++ {
			out.vocabMask[gDigit0+d] = 1
		}
	}
	if len(digits) > 0 {
		out.vocabMask[gEnd] = 1
	}
	return nil
}
