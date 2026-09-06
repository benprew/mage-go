package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestRandomSource_ScriptedResults(t *testing.T) {
	rs := NewRandomSource()
	rs.SetRandomResults([]int{10, -2, 3})

	if got := rs.RandIntn(4); got != 2 { // 10 % 4 = 2
		t.Errorf("RandIntn(4) = %d, want 2", got)
	}
	if got := rs.RandIntn(5); got != 3 { // -2 % 5 = 3
		t.Errorf("RandIntn(5) = %d, want 3", got)
	}
	if got := rs.RandIntn(2); got != 1 { // 3 % 2 = 1
		t.Errorf("RandIntn(2) = %d, want 1", got)
	}
}

func TestRandomSource_FlipCoin(t *testing.T) {
	rs := NewRandomSource()
	rs.SetCoinFlipResults([]bool{true, false, true})

	p := uuid.New()
	if !rs.FlipCoin(p) {
		t.Errorf("expected flip 1 to be true")
	}
	if rs.FlipCoin(p) {
		t.Errorf("expected flip 2 to be false")
	}
	if !rs.FlipCoin(p) {
		t.Errorf("expected flip 3 to be true")
	}
}

func TestRandomSource_CloneIsolation(t *testing.T) {
	rs := NewRandomSource()
	rs.SetRandomResults([]int{1, 2})
	rs.SetCoinFlipResults([]bool{true, false})

	clone := rs.Clone()

	// Consume from original
	if rs.RandIntn(5) != 1 {
		t.Errorf("original RandIntn(5) expected 1")
	}
	if !rs.FlipCoin(uuid.New()) {
		t.Errorf("original FlipCoin expected true")
	}

	// Clone should remain untouched
	if clone.RandIntn(5) != 1 {
		t.Errorf("clone RandIntn(5) expected 1")
	}
	if !clone.FlipCoin(uuid.New()) {
		t.Errorf("clone FlipCoin expected true")
	}
}
