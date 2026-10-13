package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func optionalXAlternatives() Cost {
	return OptionalCost(EitherCost(EitherCost(RemoveXCountersFromSourceCost(Charge))), "Pay counters?")
}

func TestOptionalEitherXCounterPaymentAmount(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, available := range []int{0, 1, 3} {
			for _, x := range []int{0, 2} {
				a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
				g := NewGame(a, NewBasePlayer("B"))
				source := NewArtifact("Synthetic", "")
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, available)
				declaration := optionalXAlternatives()
				outcomes := map[uuid.UUID]bool{source.ID(): false}
				g.resolution.SetX(x)
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{declaration}, OptionalOutcomes: outcomes}, nil)
				if err != nil {
					t.Fatal(err)
				}
				offers := a.offers
				a.accept = !accept
				g.resolution.SetX(7)
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				wantOffers, removed := 0, 0
				if available >= x {
					wantOffers = 1
				}
				paid := accept && available >= x
				if paid {
					removed = x
				}
				if a.offers != offers || offers != wantOffers || g.LastCostOptionalPaid(source.ID()) != paid || g.FindPermanent(source.ID()).Counters[Charge] != available-removed {
					t.Fatalf("accept=%v available=%d X=%d: offers=%d counters=%d paid=%v", accept, available, x, a.offers, g.FindPermanent(source.ID()).Counters[Charge], g.LastCostOptionalPaid(source.ID()))
				}
				if outcomes[source.ID()] {
					t.Fatal("preparation changed caller outcomes")
				}
				inner := declaration.(*optionalCost).inner.(*eitherCost)
				if inner.locked || inner.options[0].(*eitherCost).locked {
					t.Fatal("declaration changed")
				}
				if _, ok := inner.options[0].(*eitherCost).options[0].(*removeXCountersCost); !ok {
					t.Fatal("declaration lost X")
				}
			}
		}
	}
}

func TestOptionalEitherXCounterPaymentPreflight(t *testing.T) {
	for _, change := range []string{"return", "absent", "phase", "short", "combined"} {
		t.Run(change, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Synthetic", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
			a.ManaPool().Add(Red, 1)
			g.resolution.SetX(2)
			costs := []Cost{LifePayCost(2), optionalXAlternatives()}
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
			events := 0
			watcher := NewArtifact("Watcher", "", WithAbility(NewTriggered(EvtLifeLost, false).SetCondition(func(*GameEvent, GameReader, uuid.UUID, uuid.UUID) bool { events++; return false })))
			g.PutOnBattlefield(watcher, a.PlayerID())
			if err := tx.Commit(); err == nil {
				t.Fatal("changed source paid accepted optional cost")
			}
			if p := g.FindPermanent(source.ID()); p != nil && p.Counters[Charge] != remaining {
				t.Fatal("failed commit removed counters")
			}
			if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(tx.paidNumbers) != 0 || g.LastCostOptionalPaid(source.ID()) || events != 0 {
				t.Fatal("failed commit spent resources or published output")
			}
		})
	}
}

func TestOptionalEitherXCounterPaymentPresence(t *testing.T) {
	for _, x := range []int{0, 2} {
		for _, state := range []string{"present", "absent", "phased", "phase-return", "return"} {
			t.Run(fmt.Sprintf("X=%d/%s", x, state), func(t *testing.T) {
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
				g.resolution.SetX(x)
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{optionalXAlternatives()}}, nil)
				if err != nil {
					t.Fatal(err)
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
				g.resolution.SetX(9)
				err = tx.Commit()
				if state == "return" && x > 0 {
					if err == nil || g.LastCostOptionalPaid(source.ID()) {
						t.Fatal("returned source paid positive X")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				paid := x == 0 || (state != "absent" && state != "phased")
				wantOffers := 0
				if paid {
					wantOffers = 1
				}
				if a.offers != wantOffers || g.LastCostOptionalPaid(source.ID()) != paid {
					t.Fatal("wrong locked acceptance")
				}
				removed := 0
				if paid {
					removed = x
				}
				if p := g.FindPermanent(source.ID()); p != nil && p.Counters[Charge] != 3-removed {
					t.Fatal("wrong counter payment")
				}
			})
		}
	}
}

func TestOptionalEitherXCounterPaymentCloneIsolation(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	declaration := optionalXAlternatives()
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
	if g.FindPermanent(first.ID()).Counters[Charge] != 2 || g.FindPermanent(second.ID()).Counters[Charge] != 1 || clone.FindPermanent(first.ID()).Counters[Charge] != 1 || clone.FindPermanent(second.ID()).Counters[Charge] != 3 {
		t.Fatal("payment crossed instance boundary")
	}
}

func TestOptionalEitherXCounterPaymentSelection(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, branch := range []int{0, 1} {
			a := &optionalCounterBranchPlayer{countedMayPlayer: &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}, branch: branch}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Synthetic", "")
			g.PutOnBattlefield(source, a.PlayerID())
			for _, ct := range []CounterType{Charge, P1P1, Carrion} {
				g.MutablePermanent(source.ID()).AddCounter(ct, 3)
			}
			inner := EitherCost(RemoveXCountersFromSourceCost(Charge), EitherCost(RemoveXCountersFromSourceCost(P1P1), RemoveXCountersFromSourceCost(Carrion)))
			g.resolution.SetX(2)
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{OptionalCost(inner, "Pay counters?")}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			a.accept = !accept
			a.branch = 1 - branch
			g.resolution.SetX(8)
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			choices := 0
			selected := Charge
			if accept {
				choices = 1
				if branch == 1 {
					choices = 2
					selected = Carrion
				}
			}
			if a.offers != 1 || a.choices != choices {
				t.Fatal("choices repeated or skipped")
			}
			for _, ct := range []CounterType{Charge, P1P1, Carrion} {
				want := 3
				if accept && ct == selected {
					want = 1
				}
				if g.FindPermanent(source.ID()).Counters[ct] != want {
					t.Fatal("wrong selected counter payment")
				}
			}
		}
	}
}

func TestOptionalEitherXCounterPaymentSoleBranch(t *testing.T) {
	a := &optionalCounterBranchPlayer{countedMayPlayer: &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, branch: 1}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Synthetic", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Carrion, 3)
	g.resolution.SetX(2)
	cost := OptionalCost(EitherCost(RemoveXCountersFromSourceCost(Charge), EitherCost(RemoveXCountersFromSourceCost(P1P1), RemoveXCountersFromSourceCost(Carrion))), "Pay counters?")
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.offers != 1 || a.choices != 0 || g.FindPermanent(source.ID()).Counters[Carrion] != 1 {
		t.Fatal("sole payable branch was not selected automatically")
	}
}

func TestOptionalEitherXCounterPaymentAnnouncement(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Synthetic", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	g.resolution.SetX(5)
	locked, outcomes, err := lockPaymentCosts([]Cost{optionalXAlternatives()}, source.ID(), a.PlayerID(), nil, 2, cloneGameForCostPayment(g), a)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: locked, OptionalOutcomes: outcomes}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.accept = false
	g.resolution.SetX(7)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.offers != 1 || !g.LastCostOptionalPaid(source.ID()) || g.FindPermanent(source.ID()).Counters[Charge] != 1 {
		t.Fatal("payment used ambient X instead of announced X")
	}
}
