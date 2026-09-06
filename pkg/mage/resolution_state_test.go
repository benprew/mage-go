package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolutionState_BeginAndClear(t *testing.T) {
	s := NewResolutionState()
	targetID := uuid.New()
	srcID := uuid.New()

	obj := &StackObject{
		SourceID:            srcID,
		XValue:              5,
		ModeChoice:          2,
		EventAmount:         3,
		EventSourceID:       srcID,
		Targets:             []uuid.UUID{targetID},
		DamageDistribution:  map[uuid.UUID]int{targetID: 3},
		CounterDistribution: map[uuid.UUID]int{targetID: 2},
		CastZone:            ZoneHand,
	}

	cleanup := s.Begin(obj)
	if s.X() != 5 {
		t.Errorf("expected X 5, got %d", s.X())
	}
	if s.Mode() != 2 {
		t.Errorf("expected Mode 2, got %d", s.Mode())
	}
	if s.EventAmount() != 3 {
		t.Errorf("expected EventAmount 3, got %d", s.EventAmount())
	}
	if s.EventSourceID() != srcID {
		t.Errorf("expected EventSourceID %v, got %v", srcID, s.EventSourceID())
	}
	if s.ResolvingCastZone() != ZoneHand {
		t.Errorf("expected ResolvingCastZone Hand, got %v", s.ResolvingCastZone())
	}
	if len(s.ResolvingTargets()) != 1 || s.ResolvingTargets()[0] != targetID {
		t.Errorf("expected targets [%v], got %v", targetID, s.ResolvingTargets())
	}

	cleanup()

	if s.X() != 0 {
		t.Errorf("expected X reset to 0, got %d", s.X())
	}
	if s.Mode() != 0 {
		t.Errorf("expected Mode reset to 0, got %d", s.Mode())
	}
	if s.EventAmount() != 0 {
		t.Errorf("expected EventAmount reset to 0, got %d", s.EventAmount())
	}
	if s.EventSourceID() != uuid.Nil {
		t.Errorf("expected EventSourceID reset to nil, got %v", s.EventSourceID())
	}
	if s.ResolvingCastZone() != ZoneAny {
		t.Errorf("expected ResolvingCastZone reset to ZoneAny, got %v", s.ResolvingCastZone())
	}
	if len(s.ResolvingTargets()) != 0 {
		t.Errorf("expected ResolvingTargets reset to empty/nil, got %v", s.ResolvingTargets())
	}
}

func TestResolutionState_ColorOverrideScoped(t *testing.T) {
	s := NewResolutionState()
	srcID := uuid.New()
	colors := []Color{Red, Blue}

	cleanup := s.SetColorOverride(srcID, &colors)
	if s.ColorSourceID() != srcID {
		t.Errorf("expected ColorSourceID %v, got %v", srcID, s.ColorSourceID())
	}
	if s.ColorOverride() == nil || len(*s.ColorOverride()) != 2 {
		t.Fatalf("expected color override length 2, got %v", s.ColorOverride())
	}

	cleanup()
	if s.ColorSourceID() != uuid.Nil {
		t.Errorf("expected ColorSourceID restored to Nil, got %v", s.ColorSourceID())
	}
	if s.ColorOverride() != nil {
		t.Errorf("expected ColorOverride restored to nil, got %v", s.ColorOverride())
	}
}

func TestResolutionState_CloneIndependence(t *testing.T) {
	s := NewResolutionState()
	t1 := uuid.New()
	t2 := uuid.New()
	s.SetX(4)
	s.SetMode(1)
	s.SetResolvingTargets([]uuid.UUID{t1})
	s.SetDamageDistribution(map[uuid.UUID]int{t1: 4})
	s.SetCounterDistribution(map[uuid.UUID]int{t1: 2})
	colors := []Color{Green}
	s.SetColorOverride(t1, &colors)

	clone := s.Clone()

	// Mutate original
	s.SetX(10)
	s.SetResolvingTargets([]uuid.UUID{t1, t2})
	s.DamageDistribution()[t1] = 99
	s.CounterDistribution()[t1] = 88
	(*s.ColorOverride())[0] = White

	// Verify clone is unchanged
	if clone.X() != 4 {
		t.Errorf("expected clone X 4, got %d", clone.X())
	}
	if len(clone.ResolvingTargets()) != 1 || clone.ResolvingTargets()[0] != t1 {
		t.Errorf("expected clone targets [%v], got %v", t1, clone.ResolvingTargets())
	}
	if clone.DamageDistribution()[t1] != 4 {
		t.Errorf("expected clone DamageDistribution[t1] == 4, got %d", clone.DamageDistribution()[t1])
	}
	if clone.CounterDistribution()[t1] != 2 {
		t.Errorf("expected clone CounterDistribution[t1] == 2, got %d", clone.CounterDistribution()[t1])
	}
	if (*clone.ColorOverride())[0] != Green {
		t.Errorf("expected clone ColorOverride Green, got %v", (*clone.ColorOverride())[0])
	}
}
