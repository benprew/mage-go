package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestMillModifierCurrentControllerAndClone(t *testing.T) {
	g, a, b, source := controllerTestGame(t)
	g.AddMillReplacement(source.ID(), DoubleMillForOpponents(source.ID(), a.PlayerID()))
	assertScope := func(game *Game, controller, opponent uuid.UUID) {
		t.Helper()
		if got := game.ApplyMillModifiers(controller, 2); got != 2 {
			t.Fatalf("controller mill = %d, want 2", got)
		}
		if got := game.ApplyMillModifiers(opponent, 2); got != 4 {
			t.Fatalf("opponent mill = %d, want 4", got)
		}
	}
	assertScope(g, a.PlayerID(), b.PlayerID())
	clone := g.Clone()
	if err := ApplyEffect(clone, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	assertScope(clone, b.PlayerID(), a.PlayerID())
	assertScope(g, a.PlayerID(), b.PlayerID())
	changedClone := clone.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	assertScope(clone, a.PlayerID(), b.PlayerID())
	assertScope(changedClone, b.PlayerID(), a.PlayerID())
	assertScope(g, a.PlayerID(), b.PlayerID())
}
