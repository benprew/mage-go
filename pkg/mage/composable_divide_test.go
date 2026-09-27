package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestDivideValuesRoundingAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, tc := range []struct {
		n, d, down, up int
	}{
		{5, 2, 2, 3}, {6, 2, 3, 3}, {0, 2, 0, 0},
		{-5, 2, -3, -2}, {-6, 2, -3, -3}, {5, 0, 0, 0},
		{5, -2, -3, -2}, {-5, -2, 2, 3},
	} {
		if got := DivideRoundDown(Fixed(tc.n), Fixed(tc.d)).resolveValue(ctx); got != tc.down {
			t.Fatalf("down(%d, %d) = %d, want %d", tc.n, tc.d, got, tc.down)
		}
		if got := DivideRoundUp(Fixed(tc.n), Fixed(tc.d)).resolveValue(ctx); got != tc.up {
			t.Fatalf("up(%d, %d) = %d, want %d", tc.n, tc.d, got, tc.up)
		}
	}
	if got := DivideRoundUp(HandSizeOf(You), Fixed(2)).resolveValue(ctx); got != 0 {
		t.Fatalf("empty hand result = %d", got)
	}
	if got := DivideRoundUp(Fixed(5), HandSizeOf(You)).resolveValue(ctx); got != 0 {
		t.Fatalf("dynamic zero divisor result = %d", got)
	}
}

func TestDivideValuesBindingAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("numerator")
	d := BindNumber("denominator")
	body := Sequence(SnapshotNumber(Fixed(5), n), SnapshotNumber(Fixed(2), d),
		GainLifeFor(You, DivideRoundUp(FromNumber(n), FromNumber(d))))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("division did not use the clone's resolution bindings")
	}
}

func TestDivideValuesDeclaration(t *testing.T) {
	n := BindNumber("n")
	valid := NewInstant("Valid", "", Spell(Sequence(SnapshotNumber(Fixed(5), n),
		GainLifeFor(You, DivideRoundDown(FromNumber(n), Fixed(2))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, value := range []ResolutionValue{
		DivideRoundDown(nil, Fixed(2)),
		DivideRoundUp(Fixed(2), nil),
		DivideRoundDown(FromNumber(n), Fixed(2)),
		DivideRoundUp(Fixed(2), Fixed(0)),
	} {
		card := NewInstant("Invalid", "", Spell(GainLifeFor(You, value)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("invalid division accepted: %v", err)
		}
	}
}
