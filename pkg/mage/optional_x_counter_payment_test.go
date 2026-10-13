package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestOptionalXCounterPaymentAnnouncement(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, available := range []int{0, 1, 3} {
			for _, x := range []int{0, 2} {
				a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
				g := NewGame(a, NewBasePlayer("B"))
				source := NewArtifact("Synthetic", "")
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, available)
				paid := BindNumber("paid")
				declaration := OptionalCostInto(RemoveXCountersFromSourceCost(Charge), "Pay counters?", paid)
				g.resolution.SetX(5)
				tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), XValue: x, Costs: []Cost{declaration}})
				if err != nil {
					t.Fatal(err)
				}
				offers := a.offers
				g.resolution.SetX(7)
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				wantOffers := 0
				if available >= x {
					wantOffers = 1
				}
				if a.offers != wantOffers {
					t.Fatal("optional offer used ambient X")
				}
				wantPaid, removed := 0, 0
				if accept && available >= x {
					wantPaid, removed = 1, x
				}
				if tx.paidNumbers[paid] != wantPaid || g.FindPermanent(source.ID()).Counters[Charge] != available-removed {
					t.Fatalf("accept=%v available=%d X=%d: result=%d counters=%d", accept, available, x, tx.paidNumbers[paid], g.FindPermanent(source.ID()).Counters[Charge])
				}
				if a.offers != offers {
					t.Fatal("commit repeated choice")
				}
				if _, ok := declaration.(*optionalCost).inner.(*removeXCountersCost); !ok {
					t.Fatal("declaration changed")
				}
			}
		}
	}
}

func TestOptionalXCounterPaymentPreflight(t *testing.T) {
	for _, change := range []string{"return", "absent", "phase", "short", "combined"} {
		t.Run(change, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Synthetic", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
			a.ManaPool().Add(Red, 1)
			paid := BindNumber("paid")
			costs := []Cost{ManaPayment("{R}"), LifePayCost(2), OptionalCostInto(RemoveXCountersFromSourceCost(Charge), "Pay counters?", paid)}
			if change == "combined" {
				costs = append(costs, RemoveCountersCost(Charge, 2))
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), XValue: 2, Costs: costs})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "phase":
				g.PhaseOut(source.ID())
			case "short":
				g.MutablePermanent(source.ID()).RemoveCounter(Charge, 3)
			case "combined":
				g.MutablePermanent(source.ID()).RemoveCounter(Charge, 1)
			}
			if err := tx.Commit(); err == nil {
				t.Fatal("changed source paid accepted optional cost")
			}
			if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(tx.paidNumbers) != 0 {
				t.Fatal("failed commit spent resources or published output")
			}
		})
	}
}

func TestOptionalXCounterPaymentCloneIsolation(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	declaration := OptionalCostInto(RemoveXCountersFromSourceCost(Charge), "Pay counters?", BindNumber("paid"))
	for _, source := range []Card{first, second} {
		g.PutOnBattlefield(source, a.PlayerID())
		g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	}
	prepare := func(game *Game, source Card, x int) *costPaymentTransaction {
		tx, err := game.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), XValue: x, Costs: []Cost{declaration}})
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	one, two := prepare(g, first, 1), prepare(g, second, 2)
	clone := g.Clone()
	locked, outcomes, err := lockPaymentCosts([]Cost{declaration}, first.ID(), a.PlayerID(), nil, 2, clone, a)
	if err != nil {
		t.Fatal(err)
	}
	cloned, err := clone.prepareCostPaymentTransaction(costPaymentSpec{SourceID: first.ID(), Controller: a.PlayerID(), Costs: locked, OptionalOutcomes: outcomes}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tx := range []*costPaymentTransaction{two, cloned, one} {
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	if g.FindPermanent(first.ID()).Counters[Charge] != 2 || g.FindPermanent(second.ID()).Counters[Charge] != 1 || clone.FindPermanent(first.ID()).Counters[Charge] != 1 || clone.FindPermanent(second.ID()).Counters[Charge] != 3 {
		t.Fatal("payment crossed instance boundary")
	}
}
