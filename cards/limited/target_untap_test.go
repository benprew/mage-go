package limited

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestLeyDruidTypedUntap(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card, err := mage.CreateCard("Ley Druid")
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.RevokeBaseAttr(AttrSummonSick)
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	g.TapPermanent(land)
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 1), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{creature.ID()}); err == nil {
		t.Fatal("accepted a creature")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(land.ID()).Tapped || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("land or tap cost incorrect")
	}
}
