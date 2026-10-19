package main

import "testing"

func TestOutsideGameNativeDecision(t *testing.T) {
	pending := &apiPending{Kind: "card_from_outside_game", Options: []apiOption{{ID: "first"}, {ID: "second"}}}
	kind, ok := pendingKindToDecisionType(pending.Kind)
	if !ok || kind != decTypeChooseTargets {
		t.Fatal("outside choice has no decision specification")
	}
	if traceKindForPending(pending) != "choice_ids" {
		t.Fatal("outside choice has no card identity trace")
	}
	selected, err := actionFromStepChoice(pending, []int64{1}, 0, 10, 10)
	if err != nil || len(selected.SelectedIDs) != 1 || selected.SelectedIDs[0] != "second" {
		t.Fatal("outside choice lost selected identity", selected, err)
	}
	if pendingEncodingKind(pending.Kind) != "card_from_library" {
		t.Fatal("outside choice changed the dense encoding schema")
	}
	if pendingEncodingKind("unknown") != "unknown" {
		t.Fatal("unknown encoding changed")
	}
	out := newSpecOut()
	emitDecisionSpec(pending, testSpecIDs(), out)
	if out.decisionType != decTypeChooseTargets || out.anchorsLen != 2 {
		t.Fatal("outside candidates have no native anchors")
	}
}
