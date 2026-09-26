package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDrawResultActualCountAndReset(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		a.AddToLibrary(NewArtifact("Card", ""))
	}
	n := BindNumber("drawn")
	body := Sequence(DrawFor(You, Fixed(3), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n)))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 2 || a.Life() != 22 || g.resolution.numberBindings[n] != 2 {
		t.Fatalf("draw count: hand=%d life=%d result=%d", len(a.Hand()), a.Life(), g.resolution.numberBindings[n])
	}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || g.resolution.numberBindings[n] != 0 {
		t.Fatal("empty draw reused an old result")
	}
	if err := DrawFor(ControllerOf(This), Fixed(1), DrawAmountInto(n)).Apply(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[n] != 0 {
		t.Fatal("absent recipient reused an old result")
	}
}

func TestDrawResultReplacementAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToLibrary(NewArtifact("Card", ""))
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replaced++ })
	n := BindNumber("drawn")
	body := Sequence(DrawFor(You, Fixed(2), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n)))
	clone := g.Clone()
	if err := body.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	replaced = 0
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 || len(a.Hand()) != 0 {
		t.Fatal("clone shared draw or result state")
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || replaced != 1 || a.DrewFromEmpty() || g.resolution.numberBindings[n] != 1 {
		t.Fatalf("replacement: life=%d replaced=%d empty=%v result=%d", a.Life(), replaced, a.DrewFromEmpty(), g.resolution.numberBindings[n])
	}
}

func TestDrawResultDefinition(t *testing.T) {
	n := BindNumber("drawn")
	valid := NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{DrawFor(You, Fixed(1), DrawAmountInto(nil)), "binding"},
		{DrawFor(You, Fixed(1), DrawAmountInto(n), DrawAmountInto(n)), "once"},
		{GainLifeFor(You, FromNumber(n)), "producer"},
		{Sequence(GainLifeFor(You, FromNumber(n)), DrawFor(You, Fixed(1), DrawAmountInto(n))), "producer"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.body))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestDrawResultTargetAndStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	target := DeclarePlayerTarget("player")
	n := BindNumber("drawn")
	body := Sequence(DrawFor(target, Fixed(2), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n)))
	ability := Spell(body, WithTargets(target))
	card := NewInstant("Synthetic draw", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{a.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copied := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if len(a.Hand()) != 2 || a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("copied draw: hand=%d A=%d B=%d", len(a.Hand()), a.Life(), b.Life())
	}
	if err := DrawFor(target, Fixed(1), DrawAmountInto(n)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[n] != 0 {
		t.Fatal("missing target reused a draw result")
	}
}
