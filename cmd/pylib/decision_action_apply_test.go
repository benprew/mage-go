package main

import (
	"testing"
)

// These tests exercise actionFromDecoderOutput against synthetic apiPending
// fixtures (the same pattern used in decision_spec_emitter_test.go). They
// don't spin up a real engine game — they verify the decoder→actionRequest
// translation that applyDecoderAction relies on. Engine-side end-to-end
// coverage lives in the Python smoke (scripts/smoke_decoder_train.py).

// build a decoder action: tokens, isPointer mask, pointer subjects.
type decoded struct {
	tokens          []int32
	isPointer       []uint8
	pointerSubjects []int32
}

// vocab step.
func v(tok int32) func(*decoded) {
	return func(d *decoded) {
		d.tokens = append(d.tokens, tok)
		d.isPointer = append(d.isPointer, 0)
		d.pointerSubjects = append(d.pointerSubjects, -1)
	}
}

// pointer step targeting subject_index s.
func p(s int32) func(*decoded) {
	return func(d *decoded) {
		d.tokens = append(d.tokens, 0) // ignored on pointer steps
		d.isPointer = append(d.isPointer, 1)
		d.pointerSubjects = append(d.pointerSubjects, s)
	}
}

func mkDecoded(steps ...func(*decoded)) decoded {
	d := decoded{}
	for _, s := range steps {
		s(&d)
	}
	return d
}

func TestApplyDecoderAction_Priority(t *testing.T) {
	pending := &apiPending{
		Kind: "priority",
		Options: []apiOption{
			{Kind: "pass"},
			{Kind: "play_land", CardID: "card-1"},
			{Kind: "play_land", CardID: "card-2"},
		},
	}
	// Anchors: subject_index 0..2 → option indices 0..2.
	anchors := []int32{0, 1, 2}
	// Decoder picks subject_index=2 (the second land).
	d := mkDecoded(v(gPriorityOpen), p(2), v(gEnd))

	got, err := actionFromDecoderOutput(decTypePriority, d.tokens, d.pointerSubjects, d.isPointer, anchors, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Kind != "play_land" || got.CardID != "card-2" {
		t.Fatalf("priority got %+v, want play_land card-2", got)
	}
}

func TestApplyDecoderAction_Priority_Pass(t *testing.T) {
	pending := &apiPending{
		Kind:    "priority",
		Options: []apiOption{{Kind: "pass"}, {Kind: "play_land", CardID: "card-x"}},
	}
	anchors := []int32{0, 1}
	d := mkDecoded(v(gPriorityOpen), p(0), v(gEnd))

	got, err := actionFromDecoderOutput(decTypePriority, d.tokens, d.pointerSubjects, d.isPointer, anchors, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Kind != "pass" {
		t.Fatalf("priority pass got %+v", got)
	}
}

func TestApplyDecoderAction_DeclareAttackers(t *testing.T) {
	pending := &apiPending{
		Kind: "attackers",
		Options: []apiOption{
			{Kind: "creature", PermanentID: "perm-A"},
			{Kind: "creature", PermanentID: "perm-B"},
			{Kind: "creature", PermanentID: "perm-C"},
		},
	}
	// Renderer anchors: 3 attacker anchors (subjects 0..2 → handles 0..2)
	// then 2 defender anchors (subjects 3..4 → handles 0..1, the player ids).
	anchors := []int32{0, 1, 2, 0, 1}
	// Attack with perm-A (subj 0) into player 1 (subj 4), and perm-C (subj 2)
	// into player 1 (subj 4).
	d := mkDecoded(
		v(gDeclareAttackersOpn),
		v(gAttack), p(0), v(gDefender), p(4),
		v(gAttack), p(2), v(gDefender), p(4),
		v(gEnd),
	)

	got, err := actionFromDecoderOutput(decTypeDeclareAttackers, d.tokens, d.pointerSubjects, d.isPointer, anchors, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := map[string]bool{"perm-A": true, "perm-C": true}
	if len(got.Attackers) != 2 {
		t.Fatalf("attackers got %v, want 2 entries", got.Attackers)
	}
	for _, a := range got.Attackers {
		if !want[a] {
			t.Fatalf("unexpected attacker %q (got %v)", a, got.Attackers)
		}
	}
}

func TestApplyDecoderAction_DeclareBlockers(t *testing.T) {
	// Two blockers; both can block attacker atk-X (the only attacker).
	pending := &apiPending{
		Kind: "blockers",
		Options: []apiOption{
			{Kind: "creature", PermanentID: "blk-1", ValidTargets: []apiTarget{{ID: "atk-X"}}},
			{Kind: "creature", PermanentID: "blk-2", ValidTargets: []apiTarget{{ID: "atk-X"}}},
		},
	}
	// Renderer anchors: 2 blocker anchors (subj 0..1 → option indices 0..1)
	// then 1 attacker anchor (subj 2 → attacker-order index 0).
	anchors := []int32{0, 1, 0}
	// Block: blk-1 blocks atk-X; blk-2 blocks atk-X.
	d := mkDecoded(
		v(gDeclareBlockersOpn),
		v(gBlock), p(0), v(gAttacker), p(2),
		v(gBlock), p(1), v(gAttacker), p(2),
		v(gEnd),
	)

	got, err := actionFromDecoderOutput(decTypeDeclareBlockers, d.tokens, d.pointerSubjects, d.isPointer, anchors, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got.Blockers) != 2 {
		t.Fatalf("blockers got %v, want 2", got.Blockers)
	}
	if got.Blockers[0].Blocker != "blk-1" || got.Blockers[0].Attacker != "atk-X" {
		t.Fatalf("blocker[0] = %+v", got.Blockers[0])
	}
	if got.Blockers[1].Blocker != "blk-2" || got.Blockers[1].Attacker != "atk-X" {
		t.Fatalf("blocker[1] = %+v", got.Blockers[1])
	}
}

func TestApplyDecoderAction_ChooseTargets(t *testing.T) {
	pending := &apiPending{
		Kind: "permanent",
		Options: []apiOption{
			{ID: "id-A"},
			{ID: "id-B"},
			{ID: "id-C"},
		},
	}
	anchors := []int32{0, 1, 2}
	d := mkDecoded(v(gChooseTargetsOpen), p(1), v(gEnd))

	got, err := actionFromDecoderOutput(decTypeChooseTargets, d.tokens, d.pointerSubjects, d.isPointer, anchors, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got.SelectedIDs) != 1 || got.SelectedIDs[0] != "id-B" {
		t.Fatalf("targets got %+v, want id-B", got)
	}
}

func TestApplyDecoderAction_May_Yes(t *testing.T) {
	pending := &apiPending{Kind: "may"}
	d := mkDecoded(v(gMayOpen), v(gYes), v(gEnd))

	got, err := actionFromDecoderOutput(decTypeMay, d.tokens, d.pointerSubjects, d.isPointer, nil, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Accepted {
		t.Fatalf("may yes got Accepted=false")
	}
}

func TestApplyDecoderAction_May_No(t *testing.T) {
	pending := &apiPending{Kind: "may"}
	d := mkDecoded(v(gMayOpen), v(gNo), v(gEnd))

	got, err := actionFromDecoderOutput(decTypeMay, d.tokens, d.pointerSubjects, d.isPointer, nil, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Accepted {
		t.Fatalf("may no got Accepted=true")
	}
}

func TestApplyDecoderAction_ChooseMode(t *testing.T) {
	pending := &apiPending{
		Kind:    "mode",
		Options: []apiOption{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}}, // 13 options
	}
	// Encode "12" as digit tokens.
	d := mkDecoded(v(gChooseModeOpen), v(gDigit0+1), v(gDigit0+2), v(gEnd))

	got, err := actionFromDecoderOutput(decTypeChooseMode, d.tokens, d.pointerSubjects, d.isPointer, nil, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.SelectedIndex != 12 {
		t.Fatalf("mode got SelectedIndex=%d, want 12", got.SelectedIndex)
	}
}

func TestApplyDecoderAction_ChooseX(t *testing.T) {
	pending := &apiPending{Kind: "number", Amount: 7}
	// Encode "5" as a single digit.
	d := mkDecoded(v(gChooseXOpen), v(gDigit0+5), v(gEnd))

	got, err := actionFromDecoderOutput(decTypeChooseX, d.tokens, d.pointerSubjects, d.isPointer, nil, pending)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.X != 5 || got.SelectedIndex != 5 {
		t.Fatalf("X got X=%d SelectedIndex=%d, want 5", got.X, got.SelectedIndex)
	}
}

func TestApplyDecoderAction_NoOp(t *testing.T) {
	got, err := actionFromDecoderOutput(decTypeNone, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("noop err: %v", err)
	}
	if got.Kind != "" || len(got.Attackers) != 0 || len(got.Blockers) != 0 {
		t.Fatalf("noop got non-empty action %+v", got)
	}
}

func TestApplyDecoderAction_PriorityOutOfRange(t *testing.T) {
	pending := &apiPending{Kind: "priority", Options: []apiOption{{Kind: "pass"}}}
	anchors := []int32{99}
	d := mkDecoded(v(gPriorityOpen), p(0), v(gEnd))
	_, err := actionFromDecoderOutput(decTypePriority, d.tokens, d.pointerSubjects, d.isPointer, anchors, pending)
	if err == nil {
		t.Fatalf("expected error for out-of-range option index")
	}
}
