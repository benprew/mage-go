package interactive

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSnapshotPublicHands(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	first := mage.NewCreature("First", "", 1, 1)
	b.AddToHand(first)
	if len(SnapshotGameState(g, 0).Opponent.Hand) != 0 {
		t.Fatal("private hand exposed")
	}
	p := g.PutOnBattlefield(mage.NewEnchantment("Public hands", "", mage.WithAbility(mage.Static(mage.RevealAllHands()))), a.PlayerID())
	g.ApplyContinuousEffects()
	snap := SnapshotGameState(g, 0)
	if len(snap.Opponent.Hand) != 1 || snap.Opponent.Hand[0].ID != first.ID() {
		t.Fatal("public hand concealed")
	}
	clone := g.Clone()
	next := mage.NewArtifact("Next", "")
	b.AddToHand(next)
	snap = SnapshotGameState(g, 0)
	if len(snap.Opponent.Hand) != 2 || len(SnapshotGameState(clone, 0).Opponent.Hand) != 1 {
		t.Fatal("contents or clone visibility stale")
	}
	b.RemoveFromHand(first.ID())
	if snap = SnapshotGameState(g, 0); len(snap.Opponent.Hand) != 1 || snap.Opponent.Hand[0].ID != next.ID() {
		t.Fatal("departed hand card still exposed")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if len(SnapshotGameState(g, 0).Opponent.Hand) != 0 || SnapshotGameState(g, 0).Opponent.HandCount != 1 {
		t.Fatal("ended visibility exposed hand")
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(SnapshotGameState(g, 0).Opponent.Hand) != 1 {
		t.Fatal("cleanup did not restore visibility")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.ApplyContinuousEffects()
	if len(SnapshotGameState(g, 0).Opponent.Hand) != 0 || len(SnapshotGameState(clone, 0).Opponent.Hand) != 1 {
		t.Fatal("source removal crossed clones")
	}
}
