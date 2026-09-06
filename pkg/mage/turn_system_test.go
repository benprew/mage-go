package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTurnSystem_BasicState(t *testing.T) {
	ts := NewTurnSystem()

	if ts.Turn() != 1 {
		t.Errorf("Turn(): got %d, want 1", ts.Turn())
	}
	ts.IncrementTurn()
	if ts.Turn() != 2 {
		t.Errorf("Turn() after increment: got %d, want 2", ts.Turn())
	}
	ts.SetTurn(10)
	if ts.Turn() != 10 {
		t.Errorf("Turn() after SetTurn: got %d, want 10", ts.Turn())
	}

	if ts.Step() != 0 {
		t.Errorf("Step(): got %v, want 0", ts.Step())
	}
	ts.SetStep(PrecombatMain)
	if ts.Step() != PrecombatMain {
		t.Errorf("Step() after SetStep: got %v, want PrecombatMain", ts.Step())
	}

	if ts.ActivePlayerIndex() != 0 {
		t.Errorf("ActivePlayerIndex(): got %d, want 0", ts.ActivePlayerIndex())
	}
	ts.SetActivePlayerIndex(1)
	if ts.ActivePlayerIndex() != 1 {
		t.Errorf("ActivePlayerIndex() after SetActivePlayerIndex: got %d, want 1", ts.ActivePlayerIndex())
	}
}

func TestTurnSystem_ExtraTurns(t *testing.T) {
	ts := NewTurnSystem()
	pA := uuid.New()
	pB := uuid.New()

	if ts.HasExtraTurns() {
		t.Error("expected no extra turns initially")
	}

	ts.GrantExtraTurn(pA)
	ts.GrantExtraTurn(pB)

	if !ts.HasExtraTurns() {
		t.Error("expected extra turns present")
	}
	if len(ts.ExtraTurns()) != 2 {
		t.Errorf("ExtraTurns length: got %d, want 2", len(ts.ExtraTurns()))
	}

	p1, ok1 := ts.PopExtraTurn()
	if !ok1 || p1 != pA {
		t.Errorf("PopExtraTurn 1: got %s, want %s", p1, pA)
	}

	p2, ok2 := ts.PopExtraTurn()
	if !ok2 || p2 != pB {
		t.Errorf("PopExtraTurn 2: got %s, want %s", p2, pB)
	}

	_, ok3 := ts.PopExtraTurn()
	if ok3 {
		t.Error("expected PopExtraTurn on empty to return false")
	}
}

func TestTurnSystem_ScheduleAndSteps(t *testing.T) {
	ts := NewTurnSystem()

	ts.BuildNextTurn()
	canonical := AllSteps()
	for _, expectedStep := range canonical {
		step, ok := ts.PopNextStep()
		if !ok || step != expectedStep {
			t.Fatalf("expected step %v, got %v (ok=%v)", expectedStep, step, ok)
		}
	}
	_, ok := ts.PopNextStep()
	if ok {
		t.Error("expected no more steps in turn")
	}

	// Step insertion and skipping
	ts.BuildNextTurn()
	ts.InsertStepAfter(PrecombatMain, PostcombatMain)
	ts.SkipNextOccurrenceOfStep(Draw)

	step, _ := ts.PopNextStep()
	if step != Untap {
		t.Errorf("expected Untap, got %v", step)
	}
	step, _ = ts.PopNextStep()
	if step != Upkeep {
		t.Errorf("expected Upkeep, got %v", step)
	}
	// Draw should have been skipped
	step, _ = ts.PopNextStep()
	if step != PrecombatMain {
		t.Errorf("expected PrecombatMain after skipped Draw, got %v", step)
	}
	step, _ = ts.PopNextStep()
	if step != PostcombatMain {
		t.Errorf("expected inserted PostcombatMain, got %v", step)
	}
}

func TestTurnSystem_CombatPhaseSkip(t *testing.T) {
	ts := NewTurnSystem()
	ts.SkipNextCombatPhase()
	ts.BuildNextTurn()

	steps := make([]PhaseStep, 0)
	for {
		step, ok := ts.PopNextStep()
		if !ok {
			break
		}
		steps = append(steps, step)
	}

	for _, s := range steps {
		if s == BeginCombat || s == DeclareAttackers || s == DeclareBlockers ||
			s == FirstStrikeDamage || s == CombatDamage || s == EndCombat {
			t.Errorf("combat step %v was not skipped", s)
		}
	}
}

func TestTurnSystem_TurnSkipping(t *testing.T) {
	ts := NewTurnSystem()
	pA := uuid.New()

	if ts.ConsumeTurnSkip(pA) {
		t.Error("expected no turn skip for pA initially")
	}

	ts.SkipNextTurnFor(pA)
	if !ts.ConsumeTurnSkip(pA) {
		t.Error("expected turn skip consumed for pA")
	}
	if ts.ConsumeTurnSkip(pA) {
		t.Error("expected turn skip already consumed")
	}
}

func TestTurnSystem_SkipNextUntap(t *testing.T) {
	ts := NewTurnSystem()
	permID := uuid.New()

	if ts.HasSkipNextUntap(permID) {
		t.Error("expected no skip next untap initially")
	}
	if ts.ConsumeSkipNextUntap(permID) {
		t.Error("expected ConsumeSkipNextUntap false when none recorded")
	}

	ts.SkipNextUntap(permID)
	ts.SkipNextUntap(permID) // 2 skips queued

	if !ts.HasSkipNextUntap(permID) {
		t.Error("expected skip next untap active")
	}

	if !ts.ConsumeSkipNextUntap(permID) {
		t.Error("expected first skip consumed")
	}
	if !ts.HasSkipNextUntap(permID) {
		t.Error("expected 1 skip remaining")
	}
	if !ts.ConsumeSkipNextUntap(permID) {
		t.Error("expected second skip consumed")
	}
	if ts.HasSkipNextUntap(permID) {
		t.Error("expected 0 skips remaining")
	}

	// RemoveSkipNextUntap
	ts.SkipNextUntap(permID)
	ts.RemoveSkipNextUntap(permID)
	if ts.HasSkipNextUntap(permID) {
		t.Error("expected skip removed")
	}
}

func TestTurnSystem_CloneIsolation(t *testing.T) {
	ts := NewTurnSystem()
	pA := uuid.New()
	permID := uuid.New()

	ts.SetTurn(3)
	ts.SetStep(DeclareAttackers)
	ts.SetActivePlayerIndex(1)
	ts.GrantExtraTurn(pA)
	ts.SkipNextUntap(permID)
	ts.SkipNextTurnFor(pA)
	ts.AppendExtraStep(EndStep)

	clone := ts.Clone()

	// Verify clone values match
	if clone.Turn() != 3 || clone.Step() != DeclareAttackers || clone.ActivePlayerIndex() != 1 {
		t.Error("clone basic fields mismatch")
	}
	if !clone.HasExtraTurns() || clone.ExtraTurns()[0] != pA {
		t.Error("clone extra turns mismatch")
	}
	if !clone.HasSkipNextUntap(permID) {
		t.Error("clone skip untap mismatch")
	}

	// Mutate clone and verify original is isolated
	clone.SetTurn(99)
	clone.SetStep(Cleanup)
	clone.SetActivePlayerIndex(0)
	clone.PopExtraTurn()
	clone.ConsumeSkipNextUntap(permID)
	clone.ConsumeTurnSkip(pA)

	if ts.Turn() != 3 {
		t.Errorf("original Turn modified: got %d, want 3", ts.Turn())
	}
	if ts.Step() != DeclareAttackers {
		t.Errorf("original Step modified: got %v, want DeclareAttackers", ts.Step())
	}
	if ts.ActivePlayerIndex() != 1 {
		t.Errorf("original ActivePlayerIndex modified: got %d, want 1", ts.ActivePlayerIndex())
	}
	if !ts.HasExtraTurns() {
		t.Error("original extra turns drained by clone")
	}
	if !ts.HasSkipNextUntap(permID) {
		t.Error("original skip untap cleared by clone")
	}
	if !ts.ConsumeTurnSkip(pA) {
		t.Error("original turn skip consumed by clone")
	}
}
