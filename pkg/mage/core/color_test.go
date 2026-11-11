package core

import (
	"slices"
	"testing"
)

func TestColors(t *testing.T) {
	want := []Color{White, Blue, Black, Red, Green}
	got := Colors()
	if !slices.Equal(got, want) {
		t.Fatalf("Colors() = %v, want %v", got, want)
	}
	got[0] = Colorless
	if !slices.Equal(Colors(), want) {
		t.Fatal("caller mutation changed the color list")
	}
}
