package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestLifeGainStaticImmediateAndRoundTrip(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Static lifegain", "", 1, 4, WithLifeGainReplacement(DoubleLifeGainForController))
	for i := range 3 {
		g.PutOnBattlefield(source, a.PlayerID())
		if got := g.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 4 {
			t.Fatalf("entry %d: lifegain = %d, want 4", i, got)
		}
		if len(g.triggers.Pending()) != 0 || g.stack.Size() != 0 {
			t.Fatal("static replacement created a trigger")
		}
		if len(g.lifeGainModifiers) != 0 {
			t.Fatal("static replacement registered a lasting modifier")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		if got := g.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 2 {
			t.Fatalf("departed source: lifegain = %d, want 2", got)
		}
	}
}

func TestLifeGainStaticAbilityLossPhasingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Static lifegain", "", 1, 4, WithLifeGainReplacement(DoubleLifeGainForController))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	clone.AddContinuousEffect(RemoveAllAbilities(source.ID(), EndOfTurn))
	if got := clone.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 2 {
		t.Fatalf("lost ability: %d", got)
	}
	if got := g.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 4 {
		t.Fatalf("original ability: %d", got)
	}
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if got := clone.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 4 {
		t.Fatalf("restored ability: %d", got)
	}
	clone.PhaseOut(source.ID())
	if got := clone.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 2 {
		t.Fatalf("phased ability: %d", got)
	}
	clone.PhaseIn([]uuid.UUID{source.ID()})
	if got := clone.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 4 {
		t.Fatalf("phased-in ability: %d", got)
	}
}

func TestLifeGainStaticFactoryCurrentContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Static lifegain", "", 1, 4, WithLifeGainReplacement(DoubleLifeGainForController))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	if err := ApplyEffect(clone, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		game   *Game
		player uuid.UUID
		want   int
	}{
		{g, a.PlayerID(), 4}, {g, b.PlayerID(), 2}, {clone, a.PlayerID(), 2}, {clone, b.PlayerID(), 4},
	} {
		if got := tc.game.ApplyLifeGainModifiers(tc.player, 2); got != tc.want {
			t.Fatalf("lifegain = %d, want %d", got, tc.want)
		}
	}
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if got := clone.ApplyLifeGainModifiers(a.PlayerID(), 2); got != 4 {
		t.Fatalf("expired control: %d", got)
	}
}

func TestLifeGainStaticDefinitionValidation(t *testing.T) {
	card := NewArtifact("Invalid lifegain", "", WithLifeGainReplacement(nil))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil lifegain factory passed validation")
	}
	card = NewArtifact("Static lifegain", "", WithLifeGainReplacement(DoubleLifeGainForController))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(card.Abilities()[0]).Role; got != "static-life-gain-replacement" {
		t.Fatalf("role = %s", got)
	}
}
