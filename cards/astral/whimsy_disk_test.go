package astral

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestWhimsyDiskRegeneration(t *testing.T) {
	g := gametest.NewTestGame(t)
	a := g.GetPlayer(gametest.PlayerA).PlayerID()
	b := g.GetPlayer(gametest.PlayerB).PlayerID()
	card := mage.NewCreature("Artifact enchantment creature", "", 2, 2)
	card.AddType(core.TypeArtifact)
	card.AddType(core.TypeEnchantment)
	card.SetOwner(b)
	g.PutOnBattlefield(card, b)
	g.AddRegenerationShield(card.ID())
	enchantment := mage.NewEnchantment("Enchantment", "")
	enchantment.SetOwner(a)
	g.PutOnBattlefield(enchantment, a)
	if err := applyWhimsyAction(g.Game, uuid.New(), a, 12); err != nil {
		t.Fatal(err)
	}
	if p := g.FindPermanent(card.ID()); p == nil || !p.Tapped {
		t.Fatal("Disk branch destroyed a regenerated permanent twice")
	}
	if g.FindPermanent(enchantment.ID()) != nil {
		t.Fatal("Disk branch did not destroy an enchantment")
	}
}
