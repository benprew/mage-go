package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDirectOptionalXCounterPaymentAmount(t *testing.T) {
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
				g.resolution.SetX(x)
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{declaration}}, nil)
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

func TestDirectOptionalXCounterPaymentPreflight(t *testing.T) {
	for _, change := range []string{"return", "absent", "phase", "short", "combined"} {
		t.Run(change, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Synthetic", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
			a.ManaPool().Add(Red, 1)
			paid := BindNumber("paid")
			g.resolution.SetX(2)
			costs := []Cost{LifePayCost(2), OptionalCostInto(RemoveXCountersFromSourceCost(Charge), "Pay counters?", paid)}
			if change == "combined" {
				costs = append(costs, RemoveCountersCost(Charge, 2))
			}
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Mana: ParseManaCost("{R}"), Costs: costs}, nil)
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
			remaining := 0
			if p := g.FindPermanent(source.ID()); p != nil {
				remaining = p.Counters[Charge]
			}
			if err := tx.Commit(); err == nil {
				t.Fatal("changed source paid accepted optional cost")
			}
			if p := g.FindPermanent(source.ID()); p != nil && p.Counters[Charge] != remaining {
				t.Fatal("failed commit removed counters")
			}
			if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(tx.paidNumbers) != 0 {
				t.Fatal("failed commit spent resources or published output")
			}
		})
	}
}

func TestDirectOptionalXCounterPaymentCloneIsolation(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	declaration := OptionalCostInto(RemoveXCountersFromSourceCost(Charge), "Pay counters?", BindNumber("paid"))
	for _, source := range []Card{first, second} {
		g.PutOnBattlefield(source, a.PlayerID())
		g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	}
	prepare := func(game *Game, source Card, x int) *costPaymentTransaction {
		game.resolution.SetX(x)
		tx, err := game.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{declaration}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	one, two := prepare(g, first, 1), prepare(g, second, 2)
	clone := g.Clone()
	clone.players[0] = &directOptionalAcceptPlayer{clone.GetPlayer(a.PlayerID())}
	cloned := prepare(clone, first, 2)
	g.resolution.SetX(7)
	clone.resolution.SetX(8)
	for _, tx := range []*costPaymentTransaction{two, cloned, one} {
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	paid := declaration.(*optionalCost).result
	for _, tx := range []*costPaymentTransaction{one, two, cloned} {
		if got, exists := tx.paidNumbers[paid]; !exists || got != 1 {
			t.Fatal("payment output crossed instance boundary")
		}
	}
	if g.FindPermanent(first.ID()).Counters[Charge] != 2 || g.FindPermanent(second.ID()).Counters[Charge] != 1 || clone.FindPermanent(first.ID()).Counters[Charge] != 1 || clone.FindPermanent(second.ID()).Counters[Charge] != 3 {
		t.Fatal("payment crossed instance boundary")
	}
}

func TestDirectOptionalXCounterPaymentAbsentAndLegacy(t *testing.T) {
	for _, captured := range []bool{false, true} {
		for _, x := range []int{0, 2} {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Absent synthetic", "")
			paid := BindNumber("paid")
			var cost = OptionalCost(RemoveXCountersFromSourceCost(Charge), "Pay counters?")
			if captured {
				cost = OptionalCostInto(RemoveXCountersFromSourceCost(Charge), "Pay counters?", paid)
			}
			g.resolution.SetX(x)
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			a.accept = false
			g.resolution.SetX(9)
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			want := 0
			if x == 0 {
				want = 1
			}
			if a.offers != want {
				t.Fatal("absent source choice was not fixed")
			}
			if captured {
				if got, exists := tx.paidNumbers[paid]; !exists || got != want {
					t.Fatal("missing or wrong result")
				}
			} else if g.LastCostOptionalPaid(source.ID()) != (want == 1) {
				t.Fatal("legacy optional outcome was not fixed")
			}
		}
	}
}

type directOptionalAcceptPlayer struct{ Player }

func (*directOptionalAcceptPlayer) ChooseMayAbility(string) bool { return true }

func TestDirectOptionalXCounterPaymentLegacyMapIsolation(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Synthetic legacy", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	declaration := OptionalCost(RemoveXCountersFromSourceCost(Charge), "Pay counters?")
	outcomes := map[uuid.UUID]bool{source.ID(): false}
	g.resolution.SetX(2)
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{declaration, LifePayCost(2)}, OptionalOutcomes: outcomes}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[source.ID()] {
		t.Fatal("preparation mutated caller outcomes")
	}
	a.accept = false
	g.resolution.SetX(9)
	g.PhaseOut(source.ID())
	g.PhaseIn([]uuid.UUID{source.ID()})
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.offers != 1 || a.Life() != 18 || g.FindPermanent(source.ID()).Counters[Charge] != 1 || !g.LastCostOptionalPaid(source.ID()) {
		t.Fatal("legacy payment did not retain prepared choice and amount")
	}
}
