package interactive

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestSnapshotDeathReturnHand(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	b.AddToHand(mage.NewArtifact("Private", ""))
	card := mage.NewCreature("Public", "", 3, 2, mage.WithAbility(mage.ReturnToHandInsteadOfDyingWithPlayLock()))
	p := g.PutOnBattlefield(card, b.PlayerID())
	g.DestroyPermanent(p)
	snap := SnapshotGameState(g, 0)
	if snap.Opponent.HandCount != 2 || len(snap.Opponent.Hand) != 1 || snap.Opponent.Hand[0].ID != card.ID() {
		t.Fatal("snapshot must show only the returned visit")
	}
	clone := g.Clone()
	b.RemoveFromHand(card.ID())
	b.AddToHand(card)
	if len(SnapshotGameState(g, 0).Opponent.Hand) != 0 || len(SnapshotGameState(clone, 0).Opponent.Hand) != 1 {
		t.Fatal("new visit or clone visibility changed")
	}
}
