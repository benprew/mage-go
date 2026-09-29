package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterPaymentCommitPreflight(t *testing.T) {
	for _, kind := range []string{"fixed", "captured", "chosen", "combined"} {
		for _, change := range []string{"return", "absent", "phase", "short"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
				g := NewGame(a, NewBasePlayer("B"))
				source := NewArtifact("Source", "")
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
				a.ManaPool().Add(Red, 1)
				paid := BindNumber("paid")
				costs := []Cost{LifePayCost(2)}
				switch kind {
				case "fixed":
					costs = append(costs, RemoveCountersCost(Charge, 2))
				case "captured":
					costs = append(costs, RemoveCountersCostInto(Charge, 2, paid))
				case "chosen":
					costs = append(costs, RemoveAnyCountersCost(Charge, paid))
				case "combined":
					costs = append(costs, RemoveCountersCost(Charge, 2), RemoveAnyCountersCost(Charge, paid))
				}
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Mana: ParseManaCost("{R}"), Costs: costs}, nil)
				if err != nil {
					t.Fatal(err)
				}
				calls := a.calls
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
					count := 1
					if kind == "combined" {
						count = 3
					}
					g.MutablePermanent(source.ID()).RemoveCounter(Charge, 4-count)
				}
				if err := tx.Commit(); err == nil {
					t.Fatal("changed source paid prepared counter cost")
				}
				if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(tx.paidNumbers) != 0 || a.calls != calls {
					t.Fatalf("failed commit spent resources or repeated choice: life=%d mana=%d captures=%v calls=%d", a.Life(), a.ManaPool().Count(Red), tx.paidNumbers, a.calls)
				}
				if p := g.FindPermanent(source.ID()); p != nil {
					expected := 4
					if change == "short" {
						expected = 1
						if kind == "combined" {
							expected = 3
						}
					}
					if int(p.Counters[Charge]) != expected {
						t.Fatal("failed commit removed counters")
					}
				}
			})
		}
	}
}

func TestCounterPaymentLockIsolationAndZero(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			first, second := NewArtifact("First", ""), NewArtifact("Second", "")
			declaration := RemoveCountersCost(Charge, count)
			prepare := func(game *Game, source Card) *costPaymentTransaction {
				tx, err := game.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{declaration}}, nil)
				if err != nil {
					t.Fatal(err)
				}
				return tx
			}
			for _, source := range []Card{first, second} {
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
			}
			firstTx, secondTx := prepare(g, first), prepare(g, second)
			clone := g.Clone()
			cloneTx := prepare(clone, first)
			g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			g.PutOnBattlefield(first, a.PlayerID())
			g.MutablePermanent(first.ID()).AddCounter(Charge, 2)
			if err := firstTx.Commit(); err == nil {
				t.Fatal("returned source paid old cost")
			}
			if err := secondTx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := cloneTx.Commit(); err != nil {
				t.Fatal(err)
			}
			if int(g.FindPermanent(first.ID()).Counters[Charge]) != 2 || int(g.FindPermanent(second.ID()).Counters[Charge]) != 2-count || int(clone.FindPermanent(first.ID()).Counters[Charge]) != 2-count {
				t.Fatal("payment crossed source or clone boundary")
			}
			if declaration.(*removeCountersCost).locked {
				t.Fatal("preparation changed declaration")
			}
		})
	}
}

func TestCounterPaymentCombinedCommit(t *testing.T) {
	a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
	g.MutablePermanent(source.ID()).AddCounter(Carrion, 1)
	paid := BindNumber("chosen")
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{RemoveCountersCost(Charge, 2), RemoveAnyCountersCost(Charge, paid), RemoveCountersCost(Carrion, 1)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()).Counters[Charge] != 0 || g.FindPermanent(source.ID()).Counters[Carrion] != 0 || tx.paidNumbers[paid] != 2 || a.calls != 1 {
		t.Fatal("combined payment changed locked count")
	}
}
