package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestCondControlsMatchingExcludesPhasedPermanents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "", 1, 1)
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	condition := CondControlsMatching(IsCreature)
	if !condition(g, a.PlayerID(), card, uuid.Nil) {
		t.Fatal("controlled creature did not satisfy condition")
	}
	if condition(g, b.PlayerID(), card, uuid.Nil) {
		t.Fatal("opponent creature satisfied condition")
	}
	g.PhaseOut(card.ID())
	if condition(g, a.PlayerID(), card, uuid.Nil) {
		t.Fatal("phased creature satisfied condition")
	}
	g.PhaseIn([]uuid.UUID{card.ID()})
	if !condition(g, a.PlayerID(), card, uuid.Nil) {
		t.Fatal("creature did not satisfy condition after phasing in")
	}
}
