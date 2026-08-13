package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestSelectTargetPlayerRequiresPlayerTarget(t *testing.T) {
	playerA := NewBasePlayer("Player A")
	playerB := NewBasePlayer("Player B")
	g := NewGame(playerA, playerB)
	selector := SelectTargetPlayer()

	got := selector.Select(g, uuid.Nil, playerA.PlayerID(), []uuid.UUID{playerB.PlayerID()})
	if len(got) != 1 || got[0] != playerB.PlayerID() {
		t.Fatalf("SelectTargetPlayer() = %v, want [%s]", got, playerB.PlayerID())
	}

	permanent := g.PutOnBattlefield(NewCreature("Target Creature", "{1}{G}", 2, 2), playerB.PlayerID())
	if got := selector.Select(g, uuid.Nil, playerA.PlayerID(), []uuid.UUID{permanent.ID()}); len(got) != 0 {
		t.Fatalf("SelectTargetPlayer() accepted permanent target %s", permanent.ID())
	}
}
