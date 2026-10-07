package interactive

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestSnapshotCharacteristicPT(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID())
	for _, zone := range []string{"hand", "graveyard", "exile"} {
		card := mage.NewCreature("Count", "", 7, 0, mage.WithStaticAbility(mage.ToughnessEqualsControlledCount(mage.IsLand)))
		card.SetOwner(a.PlayerID())
		switch zone {
		case "hand":
			a.AddToHand(card)
		case "graveyard":
			a.AddToGraveyard(card)
		case "exile":
			g.ExileCard(card, card.ID())
		}
	}
	snap := SnapshotGameState(g, 0)
	for _, cards := range [][]CardState{snap.You.Hand, snap.You.Graveyard, snap.You.Exile} {
		if len(cards) != 1 || cards[0].Power != 7 || cards[0].Toughness != 1 {
			t.Fatalf("characteristic snapshot: %+v", cards)
		}
	}
}
