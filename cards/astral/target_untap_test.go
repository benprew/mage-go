package astral

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestWhimsyRandomUntapMembership(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 1), b.PlayerID())
	enchantment := g.PutOnBattlefield(mage.NewEnchantment("Enchantment", ""), a.PlayerID())
	g.TapPermanent(creature)
	g.TapPermanent(enchantment)
	g.SetRandomResults([]int{0})
	if err := applyWhimsyAction(g, creature.ID(), a.PlayerID(), 1); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(creature.ID()).Tapped || !g.FindPermanent(enchantment.ID()).Tapped {
		t.Fatal("random untap used wrong filter")
	}
}
