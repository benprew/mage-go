package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSingularLifeGainResultUsesFinalOutcome(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	result := BindNumber("gained")
	body := Sequence(GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result)), LoseLifeFor(You, FromNumber(result)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("life=%d/%d", a.Life(), b.Life())
	}
	clone := g.Clone()
	if err := GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 5 || clone.GetPlayer(a.PlayerID()).Life() != 25 || a.Life() != 20 {
		t.Fatal("replacement result or clone isolation failed")
	}
	if err := GainLifeForWithResult(FromPlayer(BindPlayer("absent")), Fixed(3), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 0 {
		t.Fatal("absent recipient or clone changed a prior result")
	}
}

func TestSingularLifeGainResultDeclaration(t *testing.T) {
	result := BindNumber("gained")
	good := NewInstant("Good", "", WithAbility(Spell(Sequence(
		GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result)),
		LoseLifeFor(You, FromNumber(result)),
	))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{GainLifeForWithResult(You, Fixed(2)), "option"},
		{GainLifeForWithResult(You, Fixed(2), LifeGainedInto(nil)), "binding"},
		{GainLifeForWithResult(You, Fixed(2), LifeGainedInto(BindNumber(""))), "name"},
		{GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result), LifeGainedInto(result)), "once"},
		{Sequence(LoseLifeFor(You, FromNumber(result)), GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result))), "producer"},
	} {
		bad := NewInstant("Bad", "", WithAbility(Spell(tc.body)))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestSingularLifeGainResultNonpositive(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	result := BindNumber("gained")
	for _, amount := range []int{0, -2} {
		g.resolution.numberBindings = map[*NumberBinding]int{result: 7}
		if err := GainLifeForWithResult(You, Fixed(amount), LifeGainedInto(result)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if g.resolution.numberBindings[result] != 0 || a.Life() != 20 {
			t.Fatalf("amount %d left result %d and life %d", amount, g.resolution.numberBindings[result], a.Life())
		}
	}
}
