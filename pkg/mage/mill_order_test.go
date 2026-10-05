package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

type millOrderPlayer struct {
	*BasePlayer
	calls                        int
	invalid                      bool
	librarySizes, graveyardSizes []int
}

func (p *millOrderPlayer) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	p.calls++
	p.librarySizes = append(p.librarySizes, len(p.Library()))
	p.graveyardSizes = append(p.graveyardSizes, len(p.Graveyard()))
	if p.invalid {
		return NewArtifact("Foreign", "")
	}
	return cards[len(cards)-1]
}

func millOrderLibrary(p *BasePlayer) []Card {
	cards := []Card{NewArtifact("Top", ""), NewArtifact("Middle", ""), NewArtifact("Bottom", "")}
	for _, c := range cards {
		c.SetOwner(p.PlayerID())
	}
	p.SetLibrary(cards)
	return cards
}

func TestMillOwnerOrderAndBatch(t *testing.T) {
	for _, amount := range []int{0, 1, 2, 5} {
		for _, invalid := range []bool{false, true} {
			p := &millOrderPlayer{BasePlayer: NewBasePlayer("A"), invalid: invalid}
			g := NewGame(p, NewBasePlayer("B"))
			cards := millOrderLibrary(p.BasePlayer)
			prior := NewArtifact("Prior", "")
			prior.SetOwner(p.PlayerID())
			p.AddToGraveyard(prior)
			clone := g.Clone()
			if err := ApplyEffect(g, Mill(You, Fixed(amount)), uuid.Nil, p.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			count := min(amount, 3)
			if p.calls != max(0, count-1) {
				t.Fatalf("count %d: choices %d", count, p.calls)
			}
			if len(p.Library()) != 3-count || len(p.Graveyard()) != count+1 || p.Graveyard()[0].ID() != prior.ID() {
				t.Fatal("wrong moved set or prior order")
			}
			want := slices.Clone(cards[:count])
			if !invalid {
				slices.Reverse(want)
			}
			for i, c := range want {
				if p.Graveyard()[i+1].ID() != c.ID() {
					t.Fatal("owner order ignored")
				}
			}
			for i := range p.librarySizes {
				if p.librarySizes[i] != 3 || p.graveyardSizes[i] != 1 {
					t.Fatal("order choice saw an intermediate move")
				}
			}
			if len(clone.GetPlayer(p.PlayerID()).Library()) != 3 {
				t.Fatal("mill changed clone")
			}
		}
	}
}

func TestMillCostOrderQueriesAndRollback(t *testing.T) {
	p := &millOrderPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(p, NewBasePlayer("B"))
	cards := millOrderLibrary(p.BasePlayer)
	cost := MillCost(2)
	for range 3 {
		if !cost.CanPay(uuid.Nil, p.PlayerID(), g) {
			t.Fatal("affordable query failed")
		}
	}
	if p.calls != 0 {
		t.Fatal("query made an order choice")
	}
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: p.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(4)}}); err == nil {
		t.Fatal("short library paid")
	}
	if p.calls != 0 || p.Life() != 20 || len(p.Library()) != 3 {
		t.Fatal("failed payment changed resources or choices")
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: p.PlayerID(), Costs: []Cost{LifePayCost(2), cost}})
	if err != nil {
		t.Fatal(err)
	}
	if p.calls != 0 {
		t.Fatal("validation made live order choices")
	}
	if err := tx.commit(); err != nil {
		t.Fatal(err)
	}
	if p.calls != 1 || p.Life() != 18 || len(p.Library()) != 1 || p.Graveyard()[0].ID() != cards[1].ID() || p.Graveyard()[1].ID() != cards[0].ID() {
		t.Fatal("committed cost did not use owner order")
	}
}

func TestMillCostOrderStaleRecheck(t *testing.T) {
	p := &millOrderPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(p, NewBasePlayer("B"))
	millOrderLibrary(p.BasePlayer)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: p.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	p.SetLibrary(p.Library()[:1])
	if err := tx.commit(); err == nil {
		t.Fatal("stale library paid")
	}
	if p.calls != 0 || p.Life() != 20 || len(p.Library()) != 1 || len(p.Graveyard()) != 0 {
		t.Fatal("stale payment did not roll back")
	}
}
