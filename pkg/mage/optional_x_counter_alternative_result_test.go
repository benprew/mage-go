package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestOptionalXCounterAlternativeResultPayment(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, available := range []int{0, 1, 4} {
			for _, count := range []int{0, 2} {
				t.Run(fmt.Sprintf("%v/%d/%d", accept, available, count), func(t *testing.T) {
					a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
					g := NewGame(a, NewBasePlayer("B"))
					source := g.PutOnBattlefield(NewArtifact("Synthetic", ""), a.PlayerID())
					g.MutablePermanent(source.ID()).AddCounter(Charge, available)
					g.resolution.SetX(count)
					paid := BindNumber("paid")
					inner := EitherCost(EitherCost(RemoveXCountersFromSourceCost(Charge)))
					cost := OptionalCostInto(inner, "Pay counters?", paid)
					tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
					if err != nil {
						t.Fatal(err)
					}
					g.resolution.SetX(9)
					a.accept = !accept
					if err := tx.Commit(); err != nil {
						t.Fatal(err)
					}
					want, offers := 0, 0
					if available >= count {
						offers = 1
						if accept {
							want = 1
						}
					}
					got, exists := tx.paidNumbers[paid]
					if !exists || got != want || a.offers != offers || int(g.FindPermanent(source.ID()).Counters[Charge]) != available-count*want {
						t.Fatalf("result=%d exists=%v offers=%d counters=%d", got, exists, a.offers, g.FindPermanent(source.ID()).Counters[Charge])
					}
					if inner.(*eitherCost).locked || inner.(*eitherCost).options[0].(*eitherCost).options[0].(*removeXCountersCost).ct != Charge {
						t.Fatal("declaration changed")
					}
					if g.LastCostOptionalPaid(source.ID()) {
						t.Fatal("typed result wrote legacy global flag")
					}
				})
			}
		}
	}
}

func TestOptionalXCounterAlternativeResultIdentityAndRollback(t *testing.T) {
	for _, state := range []string{"return", "absent", "phase", "short", "combined", "phase-return", "initial-absent", "initial-phase"} {
		for _, count := range []int{0, 2} {
			t.Run(fmt.Sprintf("%s/%d", state, count), func(t *testing.T) {
				a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
				g := NewGame(a, NewBasePlayer("B"))
				card := NewArtifact("Synthetic", "")
				if state != "initial-absent" {
					g.PutOnBattlefield(card, a.PlayerID())
					g.MutablePermanent(card.ID()).AddCounter(Charge, 4)
				}
				if state == "initial-phase" {
					g.PhaseOut(card.ID())
				}
				a.ManaPool().Add(Red, 1)
				g.resolution.SetX(count)
				paid := BindNumber("paid")
				costs := []Cost{LifePayCost(2), OptionalCostInto(EitherCost(EitherCost(RemoveXCountersFromSourceCost(Charge))), "Pay?", paid)}
				if state == "combined" {
					costs = append(costs, RemoveCountersCost(Charge, 2))
				}
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: card.ID(), Controller: a.PlayerID(), Mana: ParseManaCost("{R}"), Costs: costs}, nil)
				if err != nil {
					t.Fatal(err)
				}
				switch state {
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
					g.MutablePermanent(card.ID()).AddCounter(Charge, 4)
				case "absent":
					g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				case "phase":
					g.PhaseOut(card.ID())
				case "phase-return":
					g.PhaseOut(card.ID())
					g.PhaseIn([]uuid.UUID{card.ID()})
				case "short", "combined":
					g.MutablePermanent(card.ID()).RemoveCounter(Charge, 3)
				}
				events := 0
				watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtLifeLost, false,
					GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					events++
					return false
				})))
				g.PutOnBattlefield(watcher, a.PlayerID())
				remaining := 0
				if p := g.FindPermanent(card.ID()); p != nil {
					remaining = int(p.Counters[Charge])
				}
				fail := (count > 0 && (state == "return" || state == "absent" || state == "phase" || state == "short")) || state == "combined"
				err = tx.Commit()
				if (err != nil) != fail {
					t.Fatalf("commit error=%v, want failure=%v", err, fail)
				}
				if fail {
					if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(tx.paidNumbers) != 0 || events != 0 {
						t.Fatal("failed commit spent resources or published result")
					}
					if p := g.FindPermanent(card.ID()); p != nil && int(p.Counters[Charge]) != remaining {
						t.Fatal("failed commit removed counters")
					}
				} else {
					want := 1
					if count > 0 && (state == "initial-absent" || state == "initial-phase") {
						want = 0
					}
					if got, ok := tx.paidNumbers[paid]; !ok || got != want {
						t.Fatal("missing or wrong result")
					}
				}
			})
		}
	}
}

func TestOptionalXCounterAlternativeResultBranchesAndIsolation(t *testing.T) {
	a := &optionalCounterBranchPlayer{countedMayPlayer: &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, branch: 1}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	inner := EitherCost(RemoveXCountersFromSourceCost(Charge), EitherCost(RemoveXCountersFromSourceCost(P1P1), RemoveXCountersFromSourceCost(Carrion)))
	cost := OptionalCostInto(inner, "Pay?", paid)
	first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", ""), a.PlayerID())
	for _, id := range []uuid.UUID{first.ID(), second.ID()} {
		g.MutablePermanent(id).AddCounter(Charge, 2)
		g.MutablePermanent(id).AddCounter(P1P1, 3)
		g.MutablePermanent(id).AddCounter(Carrion, 4)
	}
	prepare := func(game *Game, id uuid.UUID) *costPaymentTransaction {
		game.resolution.SetX(2)
		tx, err := game.prepareCostPaymentTransaction(costPaymentSpec{SourceID: id, Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	one := prepare(g, first.ID())
	if a.choices != 2 || a.offers != 1 {
		t.Fatal("nested branch choices missing")
	}
	a.accept = false
	two := prepare(g, second.ID())
	clone := g.Clone()
	clone.players[0] = &directOptionalAcceptPlayer{clone.GetPlayer(a.PlayerID())}
	cloned := prepare(clone, second.ID())
	for _, tx := range []*costPaymentTransaction{two, cloned, one} {
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	if one.paidNumbers[paid] != 1 || two.paidNumbers[paid] != 0 || cloned.paidNumbers[paid] != 1 || a.offers != 2 || a.choices != 2 {
		t.Fatal("results or choices crossed transaction boundary")
	}
	if g.FindPermanent(first.ID()).Counters[Carrion] != 2 || g.FindPermanent(second.ID()).Counters[Charge] != 2 || clone.FindPermanent(second.ID()).Counters[Charge] != 0 || clone.FindPermanent(first.ID()).Counters[Carrion] != 4 {
		t.Fatal("payment crossed game boundary")
	}
	if inner.(*eitherCost).locked || inner.(*eitherCost).options[1].(*eitherCost).locked {
		t.Fatal("declaration changed")
	}
}

func TestOptionalXCounterAlternativeResultSelectedOuterBranch(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewArtifact("Synthetic", ""), a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
	g.resolution.SetX(2)
	paid := BindNumber("paid")
	optional := OptionalCostInto(EitherCost(RemoveXCountersFromSourceCost(Charge)), "Pay?", paid)
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), EitherCost(EitherCost(optional))}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.offers != 1 {
		t.Fatal("selected wrapper was not locked")
	}
	g.PhaseOut(source.ID())
	if err := tx.Commit(); err == nil {
		t.Fatal("phased source paid selected wrapper")
	}
	if a.Life() != 20 || len(tx.paidNumbers) != 0 {
		t.Fatal("failed commit published resources or result")
	}
}

func TestOptionalXCounterAlternativeResultAnnouncedSoleBranch(t *testing.T) {
	a := &optionalCounterBranchPlayer{countedMayPlayer: &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, branch: 1}
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewArtifact("Synthetic", ""), a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Carrion, 3)
	paid := BindNumber("paid")
	inner := EitherCost(RemoveXCountersFromSourceCost(Charge), EitherCost(RemoveXCountersFromSourceCost(P1P1), RemoveXCountersFromSourceCost(Carrion)))
	cost := OptionalCostInto(inner, "Pay?", paid)
	g.resolution.SetX(5)
	locked, outcomes, err := lockPaymentCosts([]Cost{cost}, source.ID(), a.PlayerID(), nil, 2, cloneGameForCostPayment(g), a)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: locked, OptionalOutcomes: outcomes}, nil)
	if err != nil {
		t.Fatal(err)
	}
	g.resolution.SetX(9)
	a.accept = false
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if got, ok := tx.paidNumbers[paid]; !ok || got != 1 || a.offers != 1 || a.choices != 0 || g.FindPermanent(source.ID()).Counters[Carrion] != 1 {
		t.Fatal("payment lost announced X, sole branch, or locked acceptance")
	}
	if g.LastCostOptionalPaid(source.ID()) || inner.(*eitherCost).locked {
		t.Fatal("payment changed shared state")
	}
}
