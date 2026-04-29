package main

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

func TestRenderPlanEmitsManaCostID(t *testing.T) {
	cardID := uuid.New()
	state := testRenderState(cardID)
	pending := &apiPending{
		Kind:      "priority",
		PlayerIdx: 0,
		Options: []apiOption{{
			Kind:     "cast_spell",
			CardID:   cardID.String(),
			CardName: "Grizzly Bears",
			ManaCost: "{1}{G}",
		}},
	}
	cfg := testRenderConfig(128)
	view := outputViews{
		renderPlan:         make([]int32, cfg.renderPlanCapacity),
		renderPlanLengths:  make([]int64, 1),
		renderPlanOverflow: make([]int64, 1),
	}

	if err := fillRenderPlan(0, state, pending, 0, cfg, view); err != nil {
		t.Fatalf("fillRenderPlan: %v", err)
	}
	if view.renderPlanOverflow[0] != 0 {
		t.Fatalf("render plan overflowed unexpectedly")
	}

	optionPayload, ok := firstOpcodePayload(view.renderPlan[:view.renderPlanLengths[0]], opOption)
	if !ok {
		t.Fatalf("OP_OPTION not found in plan: %v", view.renderPlan[:view.renderPlanLengths[0]])
	}
	want := manaCostIDForCost("{1}{G}")
	if want < 0 {
		t.Fatalf("registered mana costs did not include {1}{G}: %v", registeredManaCostStrings())
	}
	if got := optionPayload[3]; got != want {
		t.Fatalf("mana_cost_id = %d, want %d", got, want)
	}
}

func TestRenderPlanSoftOverflow(t *testing.T) {
	state := testRenderState(uuid.New())
	cfg := testRenderConfig(4)
	view := outputViews{
		renderPlan:         make([]int32, cfg.renderPlanCapacity),
		renderPlanLengths:  make([]int64, 1),
		renderPlanOverflow: make([]int64, 1),
	}

	if err := fillRenderPlan(0, state, &apiPending{Kind: "priority"}, 0, cfg, view); err != nil {
		t.Fatalf("fillRenderPlan: %v", err)
	}
	if view.renderPlanOverflow[0] != 1 {
		t.Fatalf("render_plan_overflow = %d, want 1", view.renderPlanOverflow[0])
	}
	if view.renderPlanLengths[0] > cfg.renderPlanCapacity {
		t.Fatalf("render_plan_lengths = %d, capacity = %d", view.renderPlanLengths[0], cfg.renderPlanCapacity)
	}
	if got := view.renderPlan[:view.renderPlanLengths[0]]; len(got) != 4 || got[0] != opOpenState || got[1] != opTurn {
		t.Fatalf("partial structural prologue = %v, want OP_OPEN_STATE then complete OP_TURN", got)
	}
}

func testRenderConfig(capacity int64) encodeConfig {
	return encodeConfig{
		maxOptions:          8,
		maxTargetsPerOption: 4,
		maxCachedChoices:    8,
		zoneSlotCount:       zoneSlotCount,
		gameInfoDim:         gameInfoDim,
		optionScalarDim:     optionScalarDim,
		targetScalarDim:     targetScalarDim,
		decisionCapacity:    8,
		emitRenderPlan:      true,
		renderPlanCapacity:  capacity,
	}
}

func testRenderState(cardID uuid.UUID) *apiGameState {
	playerA := interactive.PlayerState{
		ID:        uuid.New(),
		Name:      "A",
		Life:      20,
		HandCount: 1,
		Hand: []interactive.CardState{{
			ID:   cardID,
			Name: "Grizzly Bears",
		}},
		LibraryCount: 59,
	}
	playerB := interactive.PlayerState{
		ID:           uuid.New(),
		Name:         "B",
		Life:         20,
		LibraryCount: 60,
	}
	return &apiGameState{
		Turn:         1,
		Step:         "Precombat Main",
		ActivePlayer: "A",
		Players:      [2]interactive.PlayerState{playerA, playerB},
	}
}

func firstOpcodePayload(plan []int32, opcode int32) ([]int32, bool) {
	for cursor := 0; cursor < len(plan); {
		op := plan[cursor]
		arity, ok := testRenderPlanArity[op]
		if !ok || cursor+1+arity > len(plan) {
			return nil, false
		}
		if op == opcode {
			return plan[cursor+1 : cursor+1+arity], true
		}
		cursor += 1 + arity
	}
	return nil, false
}

var testRenderPlanArity = map[int32]int{
	opOpenState:    0,
	opCloseState:   0,
	opTurn:         2,
	opLife:         2,
	opMana:         3,
	opOpenPlayer:   1,
	opClosePlayer:  0,
	opOpenZone:     2,
	opCloseZone:    0,
	opPlaceCard:    4,
	opCounter:      2,
	opAttachedTo:   1,
	opOpenActions:  0,
	opCloseActions: 0,
	opOption:       5,
	opTarget:       3,
	opOpenDict:     0,
	opCloseDict:    0,
	opDictEntry:    1,
	opPlaceCardRef: 4,
}

func TestRenderPlanV1NoDictOpcodes(t *testing.T) {
	state := testRenderState(uuid.New())
	cfg := testRenderConfig(128)
	view := outputViews{
		renderPlan:         make([]int32, cfg.renderPlanCapacity),
		renderPlanLengths:  make([]int64, 1),
		renderPlanOverflow: make([]int64, 1),
	}
	if err := fillRenderPlan(0, state, &apiPending{Kind: "priority"}, 0, cfg, view); err != nil {
		t.Fatalf("fillRenderPlan: %v", err)
	}
	plan := view.renderPlan[:view.renderPlanLengths[0]]
	for _, op := range plan {
		switch op {
		case opOpenDict, opCloseDict, opDictEntry, opPlaceCardRef:
			t.Fatalf("v1 plan contained v2 opcode %d: %v", op, plan)
		}
	}
}

func TestRenderPlanV2EmitsDictAndRefs(t *testing.T) {
	state := testRenderState(uuid.New())
	cfg := testRenderConfig(128)
	cfg.dedupCardBodies = true
	view := outputViews{
		renderPlan:         make([]int32, cfg.renderPlanCapacity),
		renderPlanLengths:  make([]int64, 1),
		renderPlanOverflow: make([]int64, 1),
	}
	if err := fillRenderPlan(0, state, &apiPending{Kind: "priority"}, 0, cfg, view); err != nil {
		t.Fatalf("fillRenderPlan: %v", err)
	}
	plan := view.renderPlan[:view.renderPlanLengths[0]]
	if len(plan) < 2 || plan[0] != opOpenState || plan[1] != opOpenDict {
		t.Fatalf("v2 plan does not start with opOpenState, opOpenDict: %v", plan)
	}
	var sawDictEntry, sawCloseDict, sawCardRef, sawPlaceCard bool
	for cursor := 0; cursor < len(plan); {
		op := plan[cursor]
		arity, ok := testRenderPlanArity[op]
		if !ok {
			t.Fatalf("unknown opcode %d at %d", op, cursor)
		}
		switch op {
		case opDictEntry:
			sawDictEntry = true
		case opCloseDict:
			sawCloseDict = true
		case opPlaceCardRef:
			sawCardRef = true
		case opPlaceCard:
			sawPlaceCard = true
		}
		cursor += 1 + arity
	}
	if !sawDictEntry || !sawCloseDict {
		t.Fatalf("expected dict entry + close, got plan=%v", plan)
	}
	if !sawCardRef {
		t.Fatalf("expected opPlaceCardRef in v2 plan, got %v", plan)
	}
	if sawPlaceCard {
		t.Fatalf("v2 plan should not contain opPlaceCard, got %v", plan)
	}
}
