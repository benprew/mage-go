package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestOptionalEitherFixedCounterPaymentAmount(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, available := range []int{0, 1, 3} {
			for _, count := range []int{0, 2} {
				a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
				g := NewGame(a, NewBasePlayer("B"))
				source := NewArtifact("Synthetic", "")
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, available)

				declaration := OptionalCost(EitherCost(EitherCost(RemoveCountersCost(Charge, count))), "Pay counters?")
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
				if g.LastCostOptionalPaid(source.ID()) != (wantPaid == 1) || int(g.FindPermanent(source.ID()).Counters[Charge]) != available-removed {
					t.Fatalf("accept=%v available=%d count=%d: result=%v counters=%d", accept, available, count, g.LastCostOptionalPaid(source.ID()), g.FindPermanent(source.ID()).Counters[Charge])
				}
				if a.offers != offers {
					t.Fatal("commit repeated choice")
				}
				if inner := declaration.(*optionalCost).inner.(*eitherCost).options[0].(*eitherCost).options[0].(*removeCountersCost); inner.locked || inner.amount != count {
					t.Fatal("declaration changed")
				}
			}
		}
	}
}

func TestOptionalEitherFixedCounterPaymentPreflight(t *testing.T) {
	for _, change := range []string{"return", "absent", "phase", "short", "combined"} {
		t.Run(change, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Synthetic", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
			a.ManaPool().Add(Red, 1)

			costs := []Cost{LifePayCost(2), OptionalCost(EitherCost(EitherCost(RemoveCountersCost(Charge, 2))), "Pay counters?")}
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
				remaining = int(p.Counters[Charge])
			}
			if err := tx.Commit(); err == nil {
				t.Fatal("changed source paid accepted optional cost")
			}
			if p := g.FindPermanent(source.ID()); p != nil && int(p.Counters[Charge]) != remaining {
				t.Fatal("failed commit removed counters")
			}
			if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(tx.paidNumbers) != 0 {
				t.Fatal("failed commit spent resources or published output")
			}
		})
	}
}

func TestOptionalEitherFixedCounterPaymentCloneIsolation(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	declaration := OptionalCost(EitherCost(EitherCost(RemoveCountersCost(Charge, 2))), "Pay counters?")
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
	if g.FindPermanent(first.ID()).Counters[Charge] != 1 || g.FindPermanent(second.ID()).Counters[Charge] != 1 || clone.FindPermanent(first.ID()).Counters[Charge] != 1 || clone.FindPermanent(second.ID()).Counters[Charge] != 3 {
		t.Fatal("payment crossed instance boundary")
	}
}

func TestOptionalEitherFixedCounterPaymentPresenceAndLegacy(t *testing.T) {
	for _, count := range []int{0, 2} {
		for _, state := range []string{"present", "absent", "phased", "phase-return", "return"} {
			t.Run(fmt.Sprintf("count=%d/%s", count, state), func(t *testing.T) {
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

				cost := OptionalCost(EitherCost(EitherCost(RemoveCountersCost(Charge, count))), "Pay counters?")

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
				if g.LastCostOptionalPaid(source.ID()) != (want == 1) {
					t.Fatal("wrong legacy acceptance flag")
				}
				if p := g.FindPermanent(source.ID()); p != nil && int(p.Counters[Charge]) != 3-count*want {
					t.Fatal("wrong counter payment")
				}
			})
		}
	}
}

func TestOptionalEitherFixedCounterPaymentBranchSelection(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, available := range []int{1, 2, 4} {
			for _, branch := range []int{0, 1} {
				a := &optionalCounterBranchPlayer{countedMayPlayer: &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}, branch: branch}
				g := NewGame(a, NewBasePlayer("B"))
				source := NewArtifact("Synthetic", "")
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, available)
				inner := EitherCost(RemoveCountersCost(Charge, 2), EitherCost(RemoveCountersCost(Charge, 3), RemoveCountersCost(Charge, 4)))
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{OptionalCost(inner, "Pay counters?")}}, nil)
				if err != nil {
					t.Fatal(err)
				}
				offers, choices, removed := 0, 0, 0
				if available >= 2 {
					offers = 1
				}
				if accept && available >= 2 {
					removed = 2
					if available == 4 {
						choices = 1
						if branch == 1 {
							choices = 2
							removed = 4
						}
					}
				}
				a.accept = !accept
				a.branch = 1 - branch
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				if a.offers != offers || a.choices != choices || int(g.FindPermanent(source.ID()).Counters[Charge]) != available-removed {
					t.Fatalf("accept=%v available=%d branch=%d: offers=%d choices=%d counters=%d", accept, available, branch, a.offers, a.choices, g.FindPermanent(source.ID()).Counters[Charge])
				}
				if inner.(*eitherCost).locked || inner.(*eitherCost).options[1].(*eitherCost).locked {
					t.Fatal("branch declaration changed")
				}
			}
		}
	}
}
