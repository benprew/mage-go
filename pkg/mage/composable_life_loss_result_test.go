package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestLoseLifeEachActualResult(t *testing.T) {
	for _, amount := range []int{-1, 0, 4, 25} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		result := BindNumber("life lost")
		body := Sequence(
			LoseLifeEach(OpponentsOf(You), Fixed(amount), LifeLostInto(result)),
			GainLifeFor(You, FromNumber(result)),
		)
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := max(0, amount)
		if got := a.Life(); got != 20+want {
			t.Errorf("amount %d: gained %d, want %d", amount, got-20, want)
		}
		if got := b.Life(); got != 20-want {
			t.Errorf("amount %d: opponent lost %d, want %d", amount, 20-got, want)
		}
	}
}

func TestLoseLifeEachResultAbsentAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("life lost")
	g.resolution.numberBindings = map[*NumberBinding]int{result: 99}
	clone := g.Clone()
	if err := LoseLifeEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), Fixed(4), LifeLostInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 0 || g.resolution.numberBindings[result] != 99 {
		t.Fatal("absent result changed the original frame or kept a stale value")
	}
}

func TestLoseLifeEachResultSumsSimultaneousLoss(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("life lost")
	if err := Sequence(
		LoseLifeEach(EachPlayer, Fixed(4), LifeLostInto(result)),
		GainLifeFor(You, FromNumber(result)),
	).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 24 || b.Life() != 16 || g.resolution.numberBindings[result] != 8 {
		t.Fatalf("life totals %d/%d, result %d; want 24/16, 8", a.Life(), b.Life(), g.resolution.numberBindings[result])
	}
}

func TestLoseLifeEachResultDefinition(t *testing.T) {
	result := BindNumber("life lost")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(
		LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(result)),
		GainLifeFor(You, FromNumber(result)),
	))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatalf("valid declaration: %v", err)
	}
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"nil binding", LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(nil)), "binding"},
		{"empty name", LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(BindNumber(""))), "name"},
		{"duplicate option", LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(result), LifeLostInto(result)), "once"},
		{"missing producer", GainLifeFor(You, FromNumber(result)), "producer"},
		{"late producer", Sequence(GainLifeFor(You, FromNumber(result)), LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(result))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), tc.body)))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error %v does not contain %q", err, tc.want)
			}
		})
	}
}
