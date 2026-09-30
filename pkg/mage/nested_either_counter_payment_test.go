package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestNestedEitherFixedCounterPaymentPreflight(t *testing.T) {
	for _, change := range []string{"return", "absent", "phase", "short", "combined"} {
		t.Run(change, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Synthetic", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 4)
			a.ManaPool().Add(Red, 1)
			costs := []Cost{LifePayCost(2), EitherCost(EitherCost(RemoveCountersCost(Charge, 2)))}
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
				t.Fatal("changed source paid selected nested cost")
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

func TestNestedEitherFixedCounterPaymentCloneIsolation(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	declaration := EitherCost(EitherCost(RemoveCountersCost(Charge, 2)))
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
	if declaration.(*eitherCost).locked || declaration.(*eitherCost).options[0].(*eitherCost).locked {
		t.Fatal("declaration changed")
	}
	if g.FindPermanent(first.ID()).Counters[Charge] != 1 || g.FindPermanent(second.ID()).Counters[Charge] != 1 || clone.FindPermanent(first.ID()).Counters[Charge] != 1 || clone.FindPermanent(second.ID()).Counters[Charge] != 3 {
		t.Fatal("payment crossed instance boundary")
	}
}

type nestedCounterChoicePlayer struct {
	*BasePlayer
	choices int
}

func (p *nestedCounterChoicePlayer) ChooseMode(_ []string, _ string) int {
	p.choices++
	return 0
}

func TestNestedEitherFixedCounterPaymentSelectionAndZero(t *testing.T) {
	for _, count := range []int{0, 2} {
		a := &nestedCounterChoicePlayer{BasePlayer: NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		source := NewArtifact("Synthetic", "")
		g.PutOnBattlefield(source, a.PlayerID())
		g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
		declaration := EitherCost(EitherCost(RemoveCountersCost(Charge, count), LifePayCost(1)), LifePayCost(2))
		tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{declaration}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		g.PhaseOut(source.ID())
		g.PhaseIn([]uuid.UUID{source.ID()})
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if a.choices != 2 || a.Life() != 20 || int(g.FindPermanent(source.ID()).Counters[Charge]) != 3-count {
			t.Fatal("nested selection changed at commit")
		}
	}
}
