package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSingularLifeLossResult(t *testing.T) {
	for _, amount := range []int{-1, 0, 4, 25} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		lost := BindNumber("lost")
		body := Sequence(LoseLifeForWithResult(You, Fixed(amount), LifeLostInto(lost)), GainLifeFor(ActivePlayer, FromNumber(lost)))
		if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := max(0, amount)
		if b.Life() != 20-want || a.Life() != 20+want {
			t.Fatalf("amount %d: life %d/%d, want %d/%d", amount, a.Life(), b.Life(), 20+want, 20-want)
		}
	}
}

func TestSingularLifeLossResultAbsentAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lost := BindNumber("lost")
	g.resolution.numberBindings = map[*NumberBinding]int{lost: 99}
	clone := g.Clone()
	if err := LoseLifeForWithResult(FromPlayer(BindPlayer("absent")), Fixed(4), LifeLostInto(lost)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[lost] != 0 || g.resolution.numberBindings[lost] != 99 {
		t.Fatal("absent result changed the original frame or kept a stale value")
	}
}

func TestSingularLifeLossResultTargetAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lost := BindNumber("lost")
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	ability := Spell(Sequence(
		LoseLifeForWithResult(target, Fixed(3), LifeLostInto(lost)),
		GainLifeFor(You, FromNumber(lost)),
	), WithTargets(target))
	card := NewSorcery("Life", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	clone := g.Clone()
	copied := clone.CopySpellOnStack(card.ID(), a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	clone.ResolveStackObject(copied)
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 17 {
		t.Fatal("copied instruction lost its target or result")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original life totals")
	}
	g.ResolveStackObject(original)
	if a.Life() != 23 || b.Life() != 17 {
		t.Fatal("original instruction lost its target or result")
	}
}

func TestSingularLifeLossResultDefinition(t *testing.T) {
	lost := BindNumber("lost")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(
		LoseLifeForWithResult(You, Fixed(4), LifeLostInto(lost)), GainLifeFor(You, FromNumber(lost))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{LoseLifeForWithResult(You, Fixed(4)), "binding"},
		{LoseLifeForWithResult(You, Fixed(4), LifeLostInto(nil)), "binding"},
		{LoseLifeForWithResult(You, Fixed(4), LifeLostInto(BindNumber(""))), "name"},
		{LoseLifeForWithResult(You, Fixed(4), LifeLostInto(lost), LifeLostInto(lost)), "once"},
		{Sequence(GainLifeFor(You, FromNumber(lost)), LoseLifeForWithResult(You, Fixed(4), LifeLostInto(lost))), "producer"},
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), tc.body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("error %v does not contain %q", err, tc.want)
		}
	}
}
