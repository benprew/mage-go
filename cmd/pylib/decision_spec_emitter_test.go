package main

import (
	"testing"

	"github.com/google/uuid"
)

// testSpecIDs returns a token-id assignment used across the decision-spec
// emitter tests. The values are arbitrary but distinct so we can identify
// each tag in assertions.
func testSpecIDs() *specTokenIDs {
	ids := &specTokenIDs{
		specOpen:      9001,
		specClose:     9002,
		decisionType:  9003,
		legalAttacker: 9004,
		legalBlocker:  9005,
		legalTarget:   9006,
		legalAction:   9007,
		forAction:     9008,
		maxValueOpen:  9009,
		maxValueClose: 9010,
		playerRef0:    9011,
		playerRef1:    9012,
	}
	for i := range ids.dtName {
		ids.dtName[i] = int32(9100 + i)
	}
	ids.stackRef = make([]int32, 16)
	for i := range ids.stackRef {
		ids.stackRef[i] = int32(9200 + i)
	}
	return ids
}

// newSpecOut allocates a generously-sized output receiver. The 0..9 single-
// digit token ids are pre-loaded into the digit table so emitDigits can be
// exercised end-to-end.
func newSpecOut() *specEmitterOut {
	out := &specEmitterOut{
		tokens:  make([]int32, 1024),
		anchors: make([]pointerAnchor, 256),
	}
	// Digit table: integer i maps to its single base-10 digit token id
	// 9300 + d for d ∈ 0..9. For multi-digit ints, concat the MSB-first
	// digit token sequence.
	digits := []int32{}
	offsets := []int32{0}
	for v := int32(0); v <= 1024; v++ {
		if v < 10 {
			digits = append(digits, 9300+v)
		} else {
			var d [11]int32
			n := 0
			for x := v; x > 0; x /= 10 {
				d[n] = 9300 + (x % 10)
				n++
			}
			for i := n - 1; i >= 0; i-- {
				digits = append(digits, d[i])
			}
		}
		offsets = append(offsets, int32(len(digits)))
	}
	out.maxValueDigits = digits
	out.maxValueDigitOffsets = offsets
	out.maxValueDigitMax = 1024
	return out
}

func liveTokens(out *specEmitterOut) []int32 {
	return out.tokens[:out.tokensLen]
}

func liveAnchors(out *specEmitterOut) []pointerAnchor {
	return out.anchors[:out.anchorsLen]
}

func TestEmitDecisionSpec_Priority(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	pending := &apiPending{
		Kind:    "priority",
		Options: []apiOption{{Kind: "pass"}, {Kind: "play_land"}, {Kind: "cast_spell"}},
	}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	want := []int32{
		ids.specOpen, ids.decisionType, ids.dtName[decTypePriority],
		ids.legalAction, ids.legalAction, ids.legalAction,
		ids.specClose,
	}
	got := liveTokens(out)
	if len(got) != len(want) {
		t.Fatalf("token len: got %d want %d (got=%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("token[%d]: got %d want %d", i, got[i], w)
		}
	}
	if out.decisionType != decTypePriority {
		t.Fatalf("decision type: got %d want %d", out.decisionType, decTypePriority)
	}
	anchors := liveAnchors(out)
	if len(anchors) != 3 {
		t.Fatalf("anchor count: got %d want 3", len(anchors))
	}
	for i, a := range anchors {
		if a.kind != anchorLegalAction || a.subjectIndex != int32(i) || a.handle != int32(i) {
			t.Fatalf("anchor[%d]: %+v", i, a)
		}
		// LegalAction anchors point at the legalAction token at position 3+i
		// (after specOpen, decisionType, dtName).
		wantPos := int32(3 + i)
		if a.tokenPosition != wantPos {
			t.Fatalf("anchor[%d] pos: got %d want %d", i, a.tokenPosition, wantPos)
		}
	}
}

func TestEmitDecisionSpec_DeclareAttackers(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	pending := &apiPending{
		Kind: "attackers",
		Options: []apiOption{
			{Kind: "attacker", PermanentUUID: uuid.New()},
			{Kind: "attacker", PermanentUUID: uuid.New()},
		},
	}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	want := []int32{
		ids.specOpen, ids.decisionType, ids.dtName[decTypeDeclareAttackers],
		ids.legalAttacker, ids.legalAttacker,
		ids.playerRef0, ids.playerRef1,
		ids.specClose,
	}
	got := liveTokens(out)
	if len(got) != len(want) {
		t.Fatalf("token len: got %d want %d (got=%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("token[%d]: got %d want %d", i, got[i], w)
		}
	}
	anchors := liveAnchors(out)
	// 2 LegalAttacker + 2 Defender = 4
	if len(anchors) != 4 {
		t.Fatalf("anchor count: got %d want 4", len(anchors))
	}
	for i := range 2 {
		if anchors[i].kind != anchorLegalAttacker || anchors[i].subjectIndex != int32(i) {
			t.Fatalf("attacker anchor[%d]: %+v", i, anchors[i])
		}
	}
	for i := range 2 {
		a := anchors[2+i]
		if a.kind != anchorDefender || a.subjectIndex != int32(i) {
			t.Fatalf("defender anchor[%d]: %+v", i, a)
		}
	}
}

func TestEmitDecisionSpec_DeclareBlockers(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	atkA := uuid.New()
	atkB := uuid.New()
	pending := &apiPending{
		Kind: "blockers",
		Options: []apiOption{
			{Kind: "blocker", ValidTargets: []apiTarget{
				{IDUUID: atkA},
				{IDUUID: atkB},
			}},
			{Kind: "blocker", ValidTargets: []apiTarget{
				{IDUUID: atkA},
			}},
		},
	}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	// Expected token stream: open, dt, dt-name, blocker, blocker,
	// attacker, attacker, close.
	wantTokens := []int32{
		ids.specOpen, ids.decisionType, ids.dtName[decTypeDeclareBlockers],
		ids.legalBlocker, ids.legalBlocker,
		ids.legalAttacker, ids.legalAttacker,
		ids.specClose,
	}
	got := liveTokens(out)
	if len(got) != len(wantTokens) {
		t.Fatalf("token len: got %d want %d (got=%v)", len(got), len(wantTokens), got)
	}
	for i, w := range wantTokens {
		if got[i] != w {
			t.Fatalf("token[%d]: got %d want %d", i, got[i], w)
		}
	}
	if out.nBlockers != 2 || out.nAttackers != 2 {
		t.Fatalf("bitmap shape: got %dx%d want 2x2", out.nBlockers, out.nAttackers)
	}
	// Edge[blocker0, atkA]=1, [blocker0, atkB]=1, [blocker1, atkA]=1, [1,B]=0
	checkEdge := func(b, a int32, want byte) {
		t.Helper()
		got := out.legalEdgeBitmap[b*out.nAttackers+a]
		if got != want {
			t.Fatalf("edge[%d,%d]: got %d want %d", b, a, got, want)
		}
	}
	checkEdge(0, 0, 1)
	checkEdge(0, 1, 1)
	checkEdge(1, 0, 1)
	checkEdge(1, 1, 0)

	anchors := liveAnchors(out)
	// 2 blocker anchors + 2 attacker anchors
	if len(anchors) != 4 {
		t.Fatalf("anchor count: got %d", len(anchors))
	}
}

func TestEmitDecisionSpec_ChooseTargets(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	pending := &apiPending{
		Kind: "permanent",
		Options: []apiOption{
			{Kind: "choice", IDUUID: uuid.New()},
			{Kind: "choice", IDUUID: uuid.New()},
			{Kind: "choice", IDUUID: uuid.New()},
		},
	}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	want := []int32{
		ids.specOpen, ids.decisionType, ids.dtName[decTypeChooseTargets],
		ids.legalTarget, ids.legalTarget, ids.legalTarget,
		ids.specClose,
	}
	got := liveTokens(out)
	if len(got) != len(want) {
		t.Fatalf("tokens: got %v want %v", got, want)
	}
	if liveAnchors(out)[0].kind != anchorLegalTarget {
		t.Fatalf("anchor kind: %v", liveAnchors(out)[0])
	}
}

func TestEmitDecisionSpec_May(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	pending := &apiPending{Kind: "may"}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	want := []int32{
		ids.specOpen, ids.decisionType, ids.dtName[decTypeMay],
		ids.specClose,
	}
	got := liveTokens(out)
	if len(got) != len(want) {
		t.Fatalf("tokens: got %v want %v", got, want)
	}
	if out.anchorsLen != 0 {
		t.Fatalf("MAY should have no anchors, got %d", out.anchorsLen)
	}
}

func TestEmitDecisionSpec_ChooseMode(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	pending := &apiPending{
		Kind:    "mode",
		Options: []apiOption{{}, {}, {}},
	}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	// max_value = len(options) = 3 => digit token "3" => 9303
	want := []int32{
		ids.specOpen, ids.decisionType, ids.dtName[decTypeChooseMode],
		ids.maxValueOpen, 9303, ids.maxValueClose,
		ids.specClose,
	}
	got := liveTokens(out)
	if len(got) != len(want) {
		t.Fatalf("tokens: got %v want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("token[%d]: got %d want %d", i, got[i], w)
		}
	}
}

func TestEmitDecisionSpec_ChooseXMultiDigit(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	pending := &apiPending{
		Kind:   "number",
		Amount: 12,
	}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	// max_value = 12 => digits "12" => 9301, 9302
	want := []int32{
		ids.specOpen, ids.decisionType, ids.dtName[decTypeChooseX],
		ids.maxValueOpen, 9301, 9302, ids.maxValueClose,
		ids.specClose,
	}
	got := liveTokens(out)
	if len(got) != len(want) {
		t.Fatalf("tokens: got %v want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("token[%d]: got %d want %d", i, got[i], w)
		}
	}
}

func TestEmitDecisionSpec_NilPendingNoOp(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	if err := emitDecisionSpec(nil, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if out.tokensLen != 0 || out.anchorsLen != 0 || out.decisionType != decTypeNone {
		t.Fatalf("expected empty output for nil pending; got tokensLen=%d anchorsLen=%d dt=%d",
			out.tokensLen, out.anchorsLen, out.decisionType)
	}
}

func TestEmitDecisionSpec_UnknownKind(t *testing.T) {
	ids := testSpecIDs()
	out := newSpecOut()
	pending := &apiPending{Kind: "mana_color"}
	if err := emitDecisionSpec(pending, ids, out); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if out.decisionType != decTypeNone {
		t.Fatalf("unknown kind should leave decisionType=None; got %d", out.decisionType)
	}
	if out.tokensLen != 0 {
		t.Fatalf("unknown kind should emit no tokens; got %d", out.tokensLen)
	}
}
