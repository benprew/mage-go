package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableArithmeticSignedValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		value ResolutionValue
		want  int
	}{
		{"subtract", SubtractValues(Fixed(-2), Fixed(3)), -5},
		{"subtract negative", SubtractValues(Fixed(2), Fixed(-3)), 5},
		{"minimum", MinValue(Fixed(-2), Fixed(-3)), -3},
		{"maximum", MaxValue(Fixed(-2), Fixed(-3)), -2},
		{"equal", MinValue(Fixed(3), Fixed(3)), 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.value.resolveValue(nil); got != test.want {
				t.Fatalf("value = %d, want %d", got, test.want)
			}
		})
	}
}

func TestComposableArithmeticCapture(t *testing.T) {
	for _, toughness := range []int{-2, 0, 1, 2, 4, 5, 8} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		c := NewCreature("Subject", "", 2, toughness)
		g.PutOnBattlefield(c, a.PlayerID())
		saved := BindPermanent("subject")
		change := MinValue(Fixed(0), MaxValue(Fixed(-4), SubtractValues(Fixed(1), ToughnessOf(From(saved)))))
		effect := Sequence(Snapshot(This, saved), Until(EndOfTurn, ModifyPT(From(saved), Fixed(4), change)))
		if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := toughness
		if toughness > 1 {
			want = max(1, toughness-4)
		}
		for range 3 {
			g.ApplyContinuousEffects()
		}
		p := g.FindPermanent(c.ID())
		if p.CurrentPower(g) != 6 || p.CurrentToughness(g) != want {
			t.Fatalf("initial toughness %d: got %d/%d, want 6/%d", toughness, p.CurrentPower(g), p.CurrentToughness(g), want)
		}
		clone := g.Clone()
		clone.effects.RemoveEndOfTurn()
		clone.ApplyContinuousEffects()
		if clone.FindPermanent(c.ID()).CurrentToughness(clone) != toughness || p.CurrentToughness(g) != want {
			t.Fatal("expiration changed the original game or retained the modification")
		}
	}
}

func TestComposableArithmeticStaticEvaluation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	value := MaxValue(Fixed(0), SubtractValues(ToughnessOf(This), Fixed(1)))
	c := NewCreature("Source", "", 2, 3, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, value, Fixed(0))))))
	g.PutOnBattlefield(c, a.PlayerID())
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 4 {
		t.Fatalf("power = %d, want 4", got)
	}
	g.FindPermanent(c.ID()).Counters[P1P1] = 1
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 6 {
		t.Fatalf("power = %d, want 6", got)
	}
}

func TestComposableArithmeticValidation(t *testing.T) {
	missing := BindPermanent("missing")
	undeclared := DeclarePermanentTarget("undeclared", IsCreature)
	for _, value := range []ResolutionValue{
		SubtractValues(Fixed(1), nil),
		MinValue(nil, Fixed(0)),
		MaxValue(Fixed(0), nil),
		MinValue(Fixed(0), MaxValue(Fixed(-4), SubtractValues(Fixed(1), ToughnessOf(From(missing))))),
		MaxValue(Fixed(0), PowerOf(undeclared)),
	} {
		c := NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, value, Fixed(0))))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatalf("accepted invalid value %T", value)
		}
	}
	value := SubtractValues(Fixed(1), ToughnessOf(From(missing)))
	c := NewCreature("Missing producer", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, value, Fixed(0))))))
	if err := ValidateCardDefinition(c.Name(), c); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("error = %v", err)
	}
}
