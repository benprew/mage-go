package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestLifePaymentBoundary(t *testing.T) {
	for _, tc := range []struct {
		name         string
		life, amount int
		want         bool
	}{
		{"exact", 3, 3, true},
		{"surplus", 4, 3, true},
		{"insufficient", 2, 3, false},
		{"zero at zero", 0, 0, true},
		{"zero below zero", -2, 0, true},
		{"positive below zero", -2, 1, false},
		{"negative cost", 3, -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			a.SetLife(tc.life)
			cost := LifePayCost(tc.amount)
			if got := cost.CanPay(uuid.Nil, a.PlayerID(), g); got != tc.want {
				t.Fatalf("CanPay = %v, want %v", got, tc.want)
			}
			if cost.CanPay(uuid.Nil, uuid.New(), g) {
				t.Fatal("absent player can pay")
			}
			if tc.want {
				if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err != nil {
					t.Fatal(err)
				}
				if a.Life() != tc.life-tc.amount {
					t.Fatalf("life = %d", a.Life())
				}
			}
		})
	}
}

func TestLifePaymentTransactionBoundary(t *testing.T) {
	for _, additional := range []int{0, 1} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		a.SetLife(3)
		tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{
			Controller: a.PlayerID(), LifeLoss: 3, Costs: []Cost{LifePayCost(additional)},
		}, nil)
		if a.Life() != 3 {
			t.Fatal("preparation changed life")
		}
		if additional > 0 {
			if err == nil {
				t.Fatal("unaffordable aggregate cost accepted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 0 {
			t.Fatalf("life = %d", a.Life())
		}
	}
}
