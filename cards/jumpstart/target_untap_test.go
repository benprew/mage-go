package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestInitiatesCompanionTypedUntap(t *testing.T) {
	for _, combat := range []bool{false, true} {
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		card, err := mage.CreateCard("Initiate's Companion")
		if err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(card, a.PlayerID())
		g.TapPermanent(g.FindPermanent(card.ID()))
		g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: card.ID(), TargetID: b.PlayerID(), PlayerID: a.PlayerID(), Flag: combat, Amount: 3})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if g.FindPermanent(card.ID()).Tapped == combat {
			t.Fatal("combat damage trigger or target was lost")
		}
	}
}
