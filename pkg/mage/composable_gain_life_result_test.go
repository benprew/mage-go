package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGainLifeEachActualResult(t *testing.T) {
	for _, amount := range []int{-1, 0, 3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		result := BindNumber("life gained")
		body := Sequence(
			GainLifeEach(EachPlayer, Fixed(amount), LifeGainedInto(result)),
			LoseLifeFor(You, FromNumber(result)),
		)
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := max(0, amount)
		if a.Life() != 20-want || b.Life() != 20+want {
			t.Fatalf("amount %d: life %d/%d", amount, a.Life(), b.Life())
		}
	}
}

func TestGainLifeEachResultReplacementAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	result := BindNumber("life gained")
	g.resolution.numberBindings = map[*NumberBinding]int{result: 99}
	clone := g.Clone()
	if err := GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 10 || g.resolution.numberBindings[result] != 99 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("replacement result or clone isolation failed")
	}
	if err := GainLifeEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), Fixed(2), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 0 {
		t.Fatal("absent selection kept a stale result")
	}
}

func TestGainLifeEachResultCanReadPriorValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("life gained")
	body := Sequence(
		GainLifeEach(OpponentsOf(You), Fixed(2), LifeGainedInto(result)),
		GainLifeEach(OpponentsOf(You), FromNumber(result), LifeGainedInto(result)),
		LoseLifeFor(You, FromNumber(result)),
	)
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 24 {
		t.Fatalf("life %d/%d, want 18/24", a.Life(), b.Life())
	}
}

func TestGainLifeEachResultDeclaration(t *testing.T) {
	result := BindNumber("life gained")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(
		GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result)),
		LoseLifeFor(You, FromNumber(result)),
	))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(nil)), "binding"},
		{GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(BindNumber(""))), "name"},
		{GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result), LifeGainedInto(result)), "once"},
		{Sequence(LoseLifeFor(You, FromNumber(result)), GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result))), "producer"},
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), tc.body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("error %v does not contain %q", err, tc.want)
		}
	}
}
