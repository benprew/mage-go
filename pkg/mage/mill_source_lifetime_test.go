package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestMillModifierSourceRoundTrip(t *testing.T) {
	for _, pruneBeforeReturn := range []bool{false, true} {
		t.Run(map[bool]string{false: "return-before-mill", true: "mill-before-return"}[pruneBeforeReturn], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewArtifact("Modifier", "")
			g.PutOnBattlefield(source, a.PlayerID())
			modifier := func(_ *Game, _ uuid.UUID, amount int) int { return amount * 2 }
			g.AddMillReplacement(source.ID(), modifier)
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			if pruneBeforeReturn && g.ApplyMillModifiers(b.PlayerID(), 2) != 2 {
				t.Fatal("departed source still modified mill")
			}
			g.PutOnBattlefield(source, a.PlayerID())
			if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 2 {
				t.Fatalf("old modifier applied to returned source: %d", got)
			}
			g.AddMillReplacement(source.ID(), modifier)
			if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 4 {
				t.Fatalf("new source registration produced %d, want 4", got)
			}
		})
	}
}

func TestMillModifierAbsentSourceRegistration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Absent", "")
	g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, amount int) int { return amount * 2 })
	g.PutOnBattlefield(source, a.PlayerID())
	if got := g.ApplyMillModifiers(b.PlayerID(), 2); got != 2 {
		t.Fatalf("absent registration attached to a later source: %d", got)
	}
}

func TestComposableMillModifierPhasingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Modifier", "")
	g.PutOnBattlefield(source, b.PlayerID())
	g.AddMillReplacement(source.ID(), DoubleMillForOpponents(source.ID(), b.PlayerID()))
	g.PhaseOut(source.ID())
	clone := g.Clone()
	assertMill := func(game *Game, want int) {
		t.Helper()
		player := game.GetPlayer(a.PlayerID())
		cards := []Card{NewArtifact("Top", ""), NewArtifact("Middle", ""), NewArtifact("Bottom", "")}
		player.SetLibrary(cards)
		before := len(player.Graveyard())
		if err := ApplyEffect(game, Mill(You, Fixed(1)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if got := len(player.Graveyard()) - before; got != want {
			t.Fatalf("milled %d cards, want %d", got, want)
		}
		if player.Library()[0].ID() != cards[want].ID() {
			t.Fatal("mill changed library order")
		}
	}
	assertMill(g, 1)
	assertMill(clone, 1)
	clone.PhaseIn([]uuid.UUID{source.ID()})
	assertMill(clone, 2)
	assertMill(g, 1)
	g.PhaseIn([]uuid.UUID{source.ID()})
	assertMill(g, 2)
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	clone.PutOnBattlefield(source, b.PlayerID())
	assertMill(clone, 1)
	assertMill(g, 2)
}
