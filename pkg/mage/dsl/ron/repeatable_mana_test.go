package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRepeatableManaPayment(t *testing.T) {
	for _, name := range []string{"Sinkhole", "Hymn to Tourach"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			conversion := mage.NewCreature("Conversion fixture", "", 1, 3, mage.WithAbility(mage.Activate(mage.Costs(mage.ManaPayment("{1}")), mage.AddMana(core.Black, 1))))
			p := g.PutOnBattlefield(conversion, a.PlayerID())
			g.TapPermanent(p)
			for range 2 {
				g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
			}
			set := "LEA"
			if name == "Hymn to Tourach" {
				set = "FEM"
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
			for range 3 {
				c := repositoryCard(t, "LEA", "Forest")
				c.SetOwner(b.PlayerID())
				b.AddToHand(c)
			}
			target := land.ID()
			if name == "Hymn to Tourach" {
				target = b.PlayerID()
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{uuid.New()}, 0); err == nil {
				t.Fatal("illegal target accepted")
			}
			if a.ManaPool().TotalMana() != 0 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed casting changed state")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("payment or stack timing")
			}
			clone := g.Clone()
			g.ResolveStack()
			if name == "Sinkhole" {
				if g.FindPermanent(land.ID()) != nil || clone.FindPermanent(land.ID()) == nil {
					t.Fatal("destruction or clone identity")
				}
			} else if len(b.Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 {
				t.Fatal("discard or clone isolation")
			}
		})
	}
}
