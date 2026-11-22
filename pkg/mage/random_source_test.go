package mage

import (
	"testing"
	"uuid"
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

func TestRandomSource_SetSeedDeterminism(t *testing.T) {
	a := NewRandomSource()
	b := NewRandomSource()
	c := NewRandomSource()
	a.SetSeed(42)
	b.SetSeed(42)
	c.SetSeed(99)

	for i := range 20 {
		ga := a.RandIntn(1000)
		gb := b.RandIntn(1000)
		if ga != gb {
			t.Fatalf("step %d: seeded RandIntn mismatch: %d != %d", i, ga, gb)
		}
	}

	p := uuid.New()
	for i := range 10 {
		if a.FlipCoin(p) != b.FlipCoin(p) {
			t.Fatalf("step %d: seeded FlipCoin mismatch", i)
		}
	}

	sA := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	sB := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	sC := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	a.Shuffle(len(sA), func(i, j int) { sA[i], sA[j] = sA[j], sA[i] })
	b.Shuffle(len(sB), func(i, j int) { sB[i], sB[j] = sB[j], sB[i] })
	c.Shuffle(len(sC), func(i, j int) { sC[i], sC[j] = sC[j], sC[i] })

	sameC := true
	for i := range sA {
		if sA[i] != sB[i] {
			t.Fatalf("seeded Shuffle mismatch at %d: %v vs %v", i, sA, sB)
		}
		if sA[i] != sC[i] {
			sameC = false
		}
	}
	if sameC {
		t.Fatalf("different seeds produced identical shuffle: %v", sA)
	}
}

func TestRandomSource_SetSeedCloneIsolation(t *testing.T) {
	orig := NewRandomSource()
	orig.SetSeed(12345)
	_ = orig.RandIntn(100)

	clone := orig.Clone()

	origVals := make([]int, 10)
	for i := range origVals {
		origVals[i] = orig.RandIntn(1000)
	}
	cloneVals := make([]int, 10)
	for i := range cloneVals {
		cloneVals[i] = clone.RandIntn(1000)
	}
	for i := range origVals {
		if origVals[i] != cloneVals[i] {
			t.Fatalf("step %d: clone diverged from original: %d != %d", i, cloneVals[i], origVals[i])
		}
	}
}

func TestRandomSource_ScriptedOverridesSeed(t *testing.T) {
	a := NewRandomSource()
	b := NewRandomSource()
	a.SetSeed(7)
	b.SetSeed(7)
	a.SetRandomResults([]int{3})

	if got := a.RandIntn(10); got != 3 {
		t.Fatalf("scripted RandIntn = %d, want 3", got)
	}
	if got, want := a.RandIntn(1000), b.RandIntn(1000); got != want {
		t.Fatalf("post-scripted seeded RandIntn = %d, want %d", got, want)
	}
}
