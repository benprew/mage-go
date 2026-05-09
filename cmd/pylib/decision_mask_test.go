package main

import (
	"testing"
)

func newMaskOut(nAnchors int) *decisionMaskOutput {
	return &decisionMaskOutput{
		vocabMask:   make([]byte, grammarVocabSize),
		pointerMask: make([]byte, nAnchors),
	}
}

func vocabSet(out *decisionMaskOutput) []int32 {
	var s []int32
	for i, b := range out.vocabMask {
		if b != 0 {
			s = append(s, int32(i))
		}
	}
	return s
}

func pointerSet(out *decisionMaskOutput) []int32 {
	var s []int32
	for i, b := range out.pointerMask {
		if b != 0 {
			s = append(s, int32(i))
		}
	}
	return s
}

func equalI32(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// step runs nextMask and returns (vocabSet, pointerSet) for assertions.
func step(t *testing.T, in *decisionMaskInput, nAnchors int) ([]int32, []int32) {
	t.Helper()
	out := newMaskOut(nAnchors)
	if err := nextMask(in, out); err != nil {
		t.Fatalf("nextMask: %v", err)
	}
	return vocabSet(out), pointerSet(out)
}

func TestNextMask_Priority(t *testing.T) {
	in := &decisionMaskInput{
		decType:       decTypePriority,
		nLegalActions: 3,
		prefixTokens:  []int32{},
		prefixPointers: []int32{},
		prefixLen:     0,
	}
	v, p := step(t, in, 8)
	if !equalI32(v, []int32{gPriorityOpen}) || len(p) != 0 {
		t.Fatalf("step0: vocab=%v pointer=%v", v, p)
	}
	in.prefixTokens = []int32{gPriorityOpen}
	in.prefixPointers = []int32{-1}
	in.prefixLen = 1
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{0, 1, 2}) {
		t.Fatalf("step1: vocab=%v pointer=%v", v, p)
	}
	in.prefixTokens = []int32{gPriorityOpen, -1}
	in.prefixPointers = []int32{-1, 1}
	in.prefixLen = 2
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gEnd}) || len(p) != 0 {
		t.Fatalf("step2: vocab=%v pointer=%v", v, p)
	}
}

func TestNextMask_DeclareAttackers(t *testing.T) {
	in := &decisionMaskInput{
		decType:         decTypeDeclareAttackers,
		nLegalAttackers: 2,
		nDefenders:      2,
		prefixTokens:    make([]int32, 16),
		prefixPointers:  make([]int32, 16),
	}
	// Step 0: only OPEN.
	v, p := step(t, in, 8)
	if !equalI32(v, []int32{gDeclareAttackersOpn}) || len(p) != 0 {
		t.Fatalf("step0: %v %v", v, p)
	}
	// After OPEN: ATTACK or END (no chosen yet, but both allowed since
	// we haven't exhausted attackers).
	in.prefixTokens[0] = gDeclareAttackersOpn
	in.prefixPointers[0] = -1
	in.prefixLen = 1
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gEnd, gAttack}) || len(p) != 0 {
		t.Fatalf("step1: %v %v", v, p)
	}
	// After OPEN ATTACK: pointer over attackers (both available).
	in.prefixTokens[1] = gAttack
	in.prefixPointers[1] = -1
	in.prefixLen = 2
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{0, 1}) {
		t.Fatalf("step2: %v %v", v, p)
	}
	// Pick attacker 0: next is DEFENDER token.
	in.prefixTokens[2] = -1 // pointer step has no vocab token; but we just record what was emitted
	in.prefixPointers[2] = 0
	in.prefixLen = 3
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gDefender}) || len(p) != 0 {
		t.Fatalf("step3: %v %v", v, p)
	}
	// After DEFENDER: defender pointer (both players legal).
	in.prefixTokens[3] = gDefender
	in.prefixPointers[3] = -1
	in.prefixLen = 4
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{0, 1}) {
		t.Fatalf("step4: %v %v", v, p)
	}
	// After defender pick: ATTACK or END (still one attacker remaining).
	in.prefixTokens[4] = -1
	in.prefixPointers[4] = 1
	in.prefixLen = 5
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gEnd, gAttack}) || len(p) != 0 {
		t.Fatalf("step5: %v %v", v, p)
	}
	// Second ATTACK + pointer: only attacker 1 still available.
	in.prefixTokens[5] = gAttack
	in.prefixPointers[5] = -1
	in.prefixLen = 6
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{1}) {
		t.Fatalf("step6: %v %v", v, p)
	}
	// After picking attacker 1, then defender pick: now all attackers chosen → only END.
	in.prefixTokens[6] = -1
	in.prefixPointers[6] = 1
	in.prefixTokens[7] = gDefender
	in.prefixPointers[7] = -1
	in.prefixTokens[8] = -1
	in.prefixPointers[8] = 0
	in.prefixLen = 9
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gEnd}) || len(p) != 0 {
		t.Fatalf("step9: %v %v", v, p)
	}
}

func TestNextMask_DeclareBlockers(t *testing.T) {
	in := &decisionMaskInput{
		decType:         decTypeDeclareBlockers,
		nLegalBlockers:  2,
		nLegalAttackers: 2,
		// Edge bitmap: blocker 0 can block attacker 0 only; blocker 1 can block both.
		legalEdgeBitmap: []byte{1, 0, 1, 1},
		prefixTokens:    make([]int32, 16),
		prefixPointers:  make([]int32, 16),
	}
	// Step 0: open.
	v, p := step(t, in, 8)
	if !equalI32(v, []int32{gDeclareBlockersOpn}) || len(p) != 0 {
		t.Fatalf("step0: %v %v", v, p)
	}
	// After OPEN: BLOCK or END.
	in.prefixTokens[0] = gDeclareBlockersOpn
	in.prefixLen = 1
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gEnd, gBlock}) {
		t.Fatalf("step1: %v %v", v, p)
	}
	// BLOCK: pointer over blockers (both available).
	in.prefixTokens[1] = gBlock
	in.prefixLen = 2
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{0, 1}) {
		t.Fatalf("step2: %v %v", v, p)
	}
	// Pick blocker 0 → next is ATTACKER token.
	in.prefixPointers[2] = 0
	in.prefixLen = 3
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gAttacker}) {
		t.Fatalf("step3: %v %v", v, p)
	}
	// ATTACKER: edge mask for blocker 0 → only attacker 0.
	in.prefixTokens[3] = gAttacker
	in.prefixLen = 4
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{0}) {
		t.Fatalf("step4: %v %v", v, p)
	}
	// After attacker pick: BLOCK or END (one blocker remaining).
	in.prefixPointers[4] = 0
	in.prefixLen = 5
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gEnd, gBlock}) {
		t.Fatalf("step5: %v %v", v, p)
	}
	// Second BLOCK: blocker 1 (only remaining).
	in.prefixTokens[5] = gBlock
	in.prefixLen = 6
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{1}) {
		t.Fatalf("step6: %v %v", v, p)
	}
	// Pick blocker 1 → ATTACKER token.
	in.prefixPointers[6] = 1
	in.prefixLen = 7
	// ATTACKER mask: blocker 1 row = [1,1] → both legal.
	in.prefixTokens[7] = gAttacker
	in.prefixLen = 8
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{0, 1}) {
		t.Fatalf("step8: %v %v", v, p)
	}
}

func TestNextMask_ChooseTargets(t *testing.T) {
	in := &decisionMaskInput{
		decType:        decTypeChooseTargets,
		nLegalTargets:  4,
		prefixTokens:   make([]int32, 4),
		prefixPointers: make([]int32, 4),
	}
	v, p := step(t, in, 8)
	if !equalI32(v, []int32{gChooseTargetsOpen}) || len(p) != 0 {
		t.Fatalf("step0: %v %v", v, p)
	}
	in.prefixTokens[0] = gChooseTargetsOpen
	in.prefixLen = 1
	v, p = step(t, in, 8)
	if len(v) != 0 || !equalI32(p, []int32{0, 1, 2, 3}) {
		t.Fatalf("step1: %v %v", v, p)
	}
	in.prefixPointers[1] = 2
	in.prefixLen = 2
	v, p = step(t, in, 8)
	if !equalI32(v, []int32{gEnd}) || len(p) != 0 {
		t.Fatalf("step2: %v %v", v, p)
	}
}

func TestNextMask_May(t *testing.T) {
	in := &decisionMaskInput{
		decType:        decTypeMay,
		prefixTokens:   make([]int32, 4),
		prefixPointers: make([]int32, 4),
	}
	v, _ := step(t, in, 0)
	if !equalI32(v, []int32{gMayOpen}) {
		t.Fatalf("step0: %v", v)
	}
	in.prefixTokens[0] = gMayOpen
	in.prefixLen = 1
	v, _ = step(t, in, 0)
	if !equalI32(v, []int32{gYes, gNo}) {
		t.Fatalf("step1: %v", v)
	}
	in.prefixTokens[1] = gYes
	in.prefixLen = 2
	v, _ = step(t, in, 0)
	if !equalI32(v, []int32{gEnd}) {
		t.Fatalf("step2: %v", v)
	}
}

func TestNextMask_ChooseModeSingleDigit(t *testing.T) {
	in := &decisionMaskInput{
		decType:        decTypeChooseMode,
		maxValue:       3,
		prefixTokens:   make([]int32, 4),
		prefixPointers: make([]int32, 4),
	}
	v, _ := step(t, in, 0)
	if !equalI32(v, []int32{gChooseModeOpen}) {
		t.Fatalf("step0: %v", v)
	}
	// After OPEN: digits 0..3 allowed (no END yet — need at least one digit).
	in.prefixTokens[0] = gChooseModeOpen
	in.prefixLen = 1
	v, _ = step(t, in, 0)
	want := []int32{gDigit0, gDigit0 + 1, gDigit0 + 2, gDigit0 + 3}
	if !equalI32(v, want) {
		t.Fatalf("step1: %v want %v", v, want)
	}
	// After digit "2": END (and any next-digit options where 2*10+d <= 3, i.e. none).
	in.prefixTokens[1] = gDigit0 + 2
	in.prefixLen = 2
	v, _ = step(t, in, 0)
	if !equalI32(v, []int32{gEnd}) {
		t.Fatalf("step2: %v", v)
	}
}

func TestNextMask_ChooseXMultiDigit(t *testing.T) {
	in := &decisionMaskInput{
		decType:        decTypeChooseX,
		maxValue:       12,
		prefixTokens:   make([]int32, 4),
		prefixPointers: make([]int32, 4),
	}
	v, _ := step(t, in, 0)
	if !equalI32(v, []int32{gChooseXOpen}) {
		t.Fatalf("step0: %v", v)
	}
	// After OPEN: digits 0..9 (since base=0 and max=12, 0*10+d ≤ 12 ⇒ d ≤ 9).
	in.prefixTokens[0] = gChooseXOpen
	in.prefixLen = 1
	v, _ = step(t, in, 0)
	wantStep1 := []int32{}
	for d := int32(0); d <= 9; d++ {
		wantStep1 = append(wantStep1, gDigit0+d)
	}
	if !equalI32(v, wantStep1) {
		t.Fatalf("step1: %v want %v", v, wantStep1)
	}
	// After digit "1": current=1, base=10; 10+d ≤ 12 ⇒ d ∈ {0,1,2}. END allowed.
	in.prefixTokens[1] = gDigit0 + 1
	in.prefixLen = 2
	v, _ = step(t, in, 0)
	want := []int32{gEnd, gDigit0, gDigit0 + 1, gDigit0 + 2}
	if !equalI32(v, want) {
		t.Fatalf("step2: %v want %v", v, want)
	}
	// After "12": current=12; base=120 > 12 ⇒ no more digits. END only.
	in.prefixTokens[2] = gDigit0 + 2
	in.prefixLen = 3
	v, _ = step(t, in, 0)
	if !equalI32(v, []int32{gEnd}) {
		t.Fatalf("step3: %v", v)
	}
}
