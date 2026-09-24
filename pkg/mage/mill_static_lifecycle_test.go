package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestMillStaticImmediateAndRoundTrip(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Static mill", "", 1, 4, WithMillReplacement(DoubleMillForOpponents))
	for i := range 3 {
		g.PutOnBattlefield(source, a.PlayerID())
		if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 4 {
			t.Fatalf("entry %d: mill = %d, want 4", i, got)
		}
		if len(g.triggers.Pending()) != 0 || g.stack.Size() != 0 {
			t.Fatal("static replacement created a trigger")
		}
		if len(g.millModifiers) != 0 {
			t.Fatal("static replacement registered a lasting modifier")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 2 {
			t.Fatalf("departed source: mill = %d, want 2", got)
		}
	}
}

func TestMillStaticAbilityLossPhasingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Static mill", "", 1, 4, WithMillReplacement(DoubleMillForOpponents))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	clone.AddContinuousEffect(RemoveAllAbilities(source.ID(), EndOfTurn))
	if got := clone.ApplyMillModifiers(b.PlayerID(), 2); got != 2 {
		t.Fatalf("lost ability: %d", got)
	}
	if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 4 {
		t.Fatalf("original ability: %d", got)
	}
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if got := clone.ApplyMillModifiers(b.PlayerID(), 2); got != 4 {
		t.Fatalf("restored ability: %d", got)
	}
	clone.PhaseOut(source.ID())
	if got := clone.ApplyMillModifiers(b.PlayerID(), 2); got != 2 {
		t.Fatalf("phased ability: %d", got)
	}
	clone.PhaseIn([]uuid.UUID{source.ID()})
	if got := clone.ApplyMillModifiers(b.PlayerID(), 2); got != 4 {
		t.Fatalf("phased-in ability: %d", got)
	}
}

func TestMillStaticFactoryCurrentContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	factory := func(_ uuid.UUID, controller uuid.UUID) MillModifierFn {
		return func(_ *Game, player uuid.UUID, amount int) int {
			if player != controller {
				return amount * 2
			}
			return amount
		}
	}
	source := NewCreature("Static mill", "", 1, 4, WithMillReplacement(factory))
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
		{g, a.PlayerID(), 2}, {g, b.PlayerID(), 4}, {clone, a.PlayerID(), 4}, {clone, b.PlayerID(), 2},
	} {
		if got := tc.game.ApplyMillModifiers(tc.player, 2); got != tc.want {
			t.Fatalf("mill = %d, want %d", got, tc.want)
		}
	}
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if got := clone.ApplyMillModifiers(b.PlayerID(), 2); got != 4 {
		t.Fatalf("expired control: %d", got)
	}
}

func TestMillStaticDefinitionValidation(t *testing.T) {
	card := NewArtifact("Invalid mill", "", WithMillReplacement(nil))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil mill factory passed validation")
	}
	card = NewArtifact("Static mill", "", WithMillReplacement(DoubleMillForOpponents))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(card.Abilities()[0]).Role; got != "static-mill-replacement" {
		t.Fatalf("role = %s", got)
	}
}

func TestMillStaticMultipleSourcesAndLegacyRegistration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "", WithMillReplacement(DoubleMillForOpponents))
	second := NewArtifact("Second", "", WithMillReplacement(DoubleMillForOpponents))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.AddMillReplacement(first.ID(), func(_ *Game, _ uuid.UUID, amount int) int { return amount + 1 })
	if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 12 {
		t.Fatalf("composed modifiers: %d, want 12", got)
	}
	for _, amount := range []int{0, -1} {
		if got := g.ApplyMillModifiers(b.PlayerID(), amount); got != amount {
			t.Fatalf("nonpositive amount: %d, want %d", got, amount)
		}
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(first.ID()))
	if got := clone.ApplyMillModifiers(b.PlayerID(), 2); got != 4 {
		t.Fatalf("clone source removal: %d, want 4", got)
	}
	clone.PutOnBattlefield(first, a.PlayerID())
	if got := clone.ApplyMillModifiers(b.PlayerID(), 2); got != 8 {
		t.Fatalf("clone source return: %d, want 8", got)
	}
	if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 12 {
		t.Fatalf("original registrations changed: %d, want 12", got)
	}
}
