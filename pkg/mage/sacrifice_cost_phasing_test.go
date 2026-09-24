package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestMatchingSacrificeCostExcludesPhasedPermanents(t *testing.T) {
	for _, cost := range []Cost{SacrificeCreatureCost(), SacrificeMatchingIncludingSourceCost(IsCreature, "sacrifice"), SacrificeNMatchingCost(2, IsCreature, "sacrifice two")} {
		t.Run(cost.Text(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first := NewCreature("First", "", 1, 1)
			first.SetOwner(a.PlayerID())
			g.PutOnBattlefield(first, a.PlayerID())
			second := NewCreature("Second", "", 1, 1)
			second.SetOwner(a.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			g.PhaseOut(first.ID())
			g.PhaseOut(second.ID())
			if cost.CanPay(uuid.Nil, a.PlayerID(), g) {
				t.Fatal("phased permanents can pay a sacrifice cost")
			}
			if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err == nil {
				t.Fatal("sacrificed a phased permanent")
			}
			g.PhaseIn([]uuid.UUID{second.ID()})
			if cost.(*sacrificeMatchingCost).count == 2 {
				if cost.CanPay(uuid.Nil, a.PlayerID(), g) {
					t.Fatal("phased permanent counted toward a two-permanent cost")
				}
				g.PhaseIn([]uuid.UUID{first.ID()})
			}
			if !cost.CanPay(uuid.Nil, a.PlayerID(), g) {
				t.Fatal("available permanents cannot pay")
			}
			if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(second.ID()) != nil {
				t.Fatal("payment did not sacrifice the available permanent")
			}
		})
	}
}
