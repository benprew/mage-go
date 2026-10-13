package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDirectOptionalFixedCounterPaymentAmount(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, available := range []int{0, 1, 3} {
			for _, count := range []int{0, 2} {
				a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
				g := NewGame(a, NewBasePlayer("B"))
				source := NewArtifact("Synthetic", "")
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, available)
				paid := BindNumber("paid")
				declaration := OptionalCostInto(RemoveCountersCost(Charge, count), "Pay counters?", paid)
				g.resolution.SetX(count)
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
				if available >= count {
					wantOffers = 1
				}
				if a.offers != wantOffers {
					t.Fatal("wrong affordability offer")
				}
				wantPaid, removed := 0, 0
				if accept && available >= count {
					wantPaid, removed = 1, count
				}
				if tx.paidNumbers[paid] != wantPaid || g.FindPermanent(source.ID()).Counters[Charge] != available-removed {
					t.Fatalf("accept=%v available=%d count=%d: result=%d counters=%d", accept, available, count, tx.paidNumbers[paid], g.FindPermanent(source.ID()).Counters[Charge])
				}
				if a.offers != offers {
					t.Fatal("commit repeated choice")
				}
				if inner := declaration.(*optionalCost).inner.(*removeCountersCost); inner.locked || inner.amount != count {
					t.Fatal("declaration changed")
				}
			}
		}
	}
}

func TestDirectOptionalFixedCounterPaymentPreflight(t *testing.T) {
	for _, change := range []string{"return", "absent", "phase", "short", "combined"} {
		t.Run(change, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Synthetic", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
			a.ManaPool().Add(Red, 1)
			paid := BindNumber("paid")
			costs := []Cost{LifePayCost(2), OptionalCostInto(RemoveCountersCost(Charge, 2), "Pay counters?", paid)}
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

func TestDirectOptionalFixedCounterPaymentCloneIsolation(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	declaration := OptionalCostInto(RemoveCountersCost(Charge, 2), "Pay counters?", BindNumber("paid"))
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
	one, two := prepare(g, first, 2), prepare(g, second, 2)
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
	if g.FindPermanent(first.ID()).Counters[Charge] != 1 || g.FindPermanent(second.ID()).Counters[Charge] != 1 || clone.FindPermanent(first.ID()).Counters[Charge] != 1 || clone.FindPermanent(second.ID()).Counters[Charge] != 3 {
		t.Fatal("payment crossed instance boundary")
	}
}

func TestDirectOptionalFixedCounterPaymentPresenceAndLegacy(t *testing.T) {
	for _, captured := range []bool{false, true} {
		for _, count := range []int{0, 2} {
			for _, state := range []string{"present", "absent", "phased", "phase-return", "return"} {
				t.Run(fmt.Sprintf("capture=%v/count=%d/%s", captured, count, state), func(t *testing.T) {
					a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
					g := NewGame(a, NewBasePlayer("B"))
					source := NewArtifact("Synthetic", "")
					if state != "absent" {
						g.PutOnBattlefield(source, a.PlayerID())
						g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
					}
					if state == "phased" {
						g.PhaseOut(source.ID())
					}
					paid := BindNumber("paid")
					cost := OptionalCost(RemoveCountersCost(Charge, count), "Pay counters?")
					if captured {
						cost = OptionalCostInto(RemoveCountersCost(Charge, count), "Pay counters?", paid)
					}
					outcomes := map[uuid.UUID]bool{source.ID(): false}
					tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}, OptionalOutcomes: outcomes}, nil)
					if err != nil {
						t.Fatal(err)
					}
					if outcomes[source.ID()] {
						t.Fatal("preparation changed caller outcomes")
					}
					if state == "phase-return" {
						g.PhaseOut(source.ID())
						g.PhaseIn([]uuid.UUID{source.ID()})
					}
					if state == "return" {
						g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
						g.PutOnBattlefield(source, a.PlayerID())
						g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
					}
					a.accept = false
					err = tx.Commit()
					if state == "return" {
						if err == nil || len(tx.paidNumbers) != 0 || g.LastCostOptionalPaid(source.ID()) {
							t.Fatal("returned source paid or published output")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					want := 1
					if state == "absent" || state == "phased" {
						want = 0
					}
					if a.offers != want {
						t.Fatal("choice did not stay locked")
					}
					if captured {
						if got, exists := tx.paidNumbers[paid]; !exists || got != want {
							t.Fatal("wrong acceptance output")
						}
					} else if g.LastCostOptionalPaid(source.ID()) != (want == 1) {
						t.Fatal("wrong legacy acceptance flag")
					}
					if p := g.FindPermanent(source.ID()); p != nil && p.Counters[Charge] != 3-count*want {
						t.Fatal("wrong counter payment")
					}
				})
			}
		}
	}
}
