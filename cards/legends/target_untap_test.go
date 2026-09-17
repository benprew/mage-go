package legends

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestResetTypedLandLoop(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	own := g.PutOnBattlefield(mage.NewLand("Own"), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 1), a.PlayerID())
	for _, p := range []*mage.Permanent{own, other, creature} {
		g.TapPermanent(p)
	}
	card, err := mage.CreateCard("Reset")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(own.ID()).Tapped || !g.FindPermanent(other.ID()).Tapped || !g.FindPermanent(creature.ID()).Tapped {
		t.Fatal("wrong Reset membership")
	}
}
