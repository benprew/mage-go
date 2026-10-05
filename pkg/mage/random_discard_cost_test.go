package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomDiscardCostPayment(t *testing.T) {
	for _, mode := range []string{"ordinary", "library", "prevented", "recheck", "returned", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			a.ManaPool().Add(Red, 1)
			g.SetRandomResults([]int{1, 7})
			if mode == "library" || mode == "recheck" {
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			}
			if mode == "prevented" {
				g.effects.AddReplacement(&preventOneCostDiscardReplacement{card: second.ID()})
			}
			if mode == "recheck" {
				g.effects.AddReplacement(&discardToHandReplacement{})
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ManaPayment("{R}"), LifePayCost(2), DiscardRandomCost(1)}})
			if mode == "prevented" || mode == "recheck" {
				if err == nil {
					t.Fatal("prevented random selection paid the cost")
				}
				if a.Life() != 20 || len(a.Hand()) != 2 || a.ManaPool().Count(Red) != 1 || g.RandIntn(10) != 1 {
					t.Fatal("failure spent resources or RNG")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(g.random.randomResults) != 2 {
				t.Fatal("preparation consumed live RNG")
			}
			if mode == "returned" {
				card, _ := a.RemoveFromHand(second.ID())
				a.AddToHand(card)
				if tx.Commit() == nil {
					t.Fatal("returned visit paid locked selection")
				}
				if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Hand()) != 2 || g.RandIntn(10) != 1 {
					t.Fatal("stale selection spent resources")
				}
				return
			}
			if mode == "clone" {
				clone := g.Clone()
				copied := *tx
				copied.game = clone
				if err := copied.Commit(); err != nil {
					t.Fatal(err)
				}
				if len(a.Hand()) != 2 || g.RandIntn(10) != 1 || clone.RandIntn(10) != 7 {
					t.Fatal("clone changed original RNG or hand")
				}
				return
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 18 || a.ManaPool().Count(Red) != 0 || len(a.Hand()) != 1 || a.Hand()[0].ID() != first.ID() || g.RandIntn(10) != 7 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 1 {
				t.Fatal("wrong random payment")
			}
			if mode == "library" && (len(a.Library()) != 1 || len(a.Graveyard()) != 0) {
				t.Fatal("replacement destination lost")
			}
		})
	}
}

func TestRandomDiscardCostOrderAndBatch(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for _, name := range []string{"Chosen", "Random one", "Random two"} {
		a.AddToHand(NewLand(name))
	}
	g.SetRandomResults([]int{1, 0, 7})
	seen := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(EvtDiscard, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, game GameReader, _, _ uuid.UUID) bool {
		if game.GetPlayer(evt.PlayerID) != a {
			return false
		}
		seen++
		if seen > 1 && len(game.GetPlayer(evt.PlayerID).Hand()) != 0 {
			t.Error("random batch event saw partial movement")
		}
		return false
	}))), a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardRandomCost(2), DiscardCost(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 3 || a.Graveyard()[0].Name() != "Chosen" || a.Graveyard()[1].Name() != "Random two" || seen != 3 || g.RandIntn(10) != 7 {
		t.Fatal("deterministic payment did not precede random batch")
	}
}

func TestRandomDiscardCostInvalidAndDirect(t *testing.T) {
	for _, amount := range []int{-1, 0, 3} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		a.AddToHand(NewLand("One"))
		g.SetRandomResults([]int{7})
		cost := DiscardRandomCost(amount)
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
		if (ValidateCardDefinition(card.Name(), card) == nil) != (amount >= 0) {
			t.Fatal("invalid count declaration accepted")
		}
		if (cost.Pay(uuid.Nil, a.PlayerID(), g) == nil) != (amount == 0) {
			t.Fatal("invalid direct payment accepted")
		}
		if len(a.Hand()) != 1 || g.RandIntn(10) != 7 {
			t.Fatal("zero or failed payment changed state")
		}
	}
}

func TestRandomDiscardCostCommitRecheck(t *testing.T) {
	for _, mode := range []string{"replacement", "RNG", "other visit", "short hand"} {
		t.Run(mode, func(t *testing.T) {
			a := &alternatingDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			a.ManaPool().Add(Red, 1)
			g.SetRandomResults([]int{1, 7})
			cost := DiscardRandomCost(1)
			for range 3 {
				if !cost.CanPay(uuid.Nil, a.PlayerID(), g) {
					t.Fatal("payable query failed")
				}
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ManaPayment("{R}"), LifePayCost(2), cost}})
			if err != nil {
				t.Fatal(err)
			}
			wantRandom := 1
			switch mode {
			case "replacement":
				g.effects.AddReplacement(&preventCostDiscardReplacement{})
			case "RNG":
				g.RandIntn(10)
				wantRandom = 7
			case "other visit":
				card, _ := a.RemoveFromHand(first.ID())
				a.AddToHand(card)
			case "short hand":
				a.RemoveFromHand(first.ID())
			}
			if err = tx.Commit(); err == nil {
				t.Fatal("changed random payment committed")
			}
			if a.choices != 0 || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 || g.RandIntn(10) != wantRandom {
				t.Fatal("recheck spent resources, chose a card, or consumed RNG")
			}
		})
	}
}

func TestRandomDiscardCostRepeatedPayments(t *testing.T) {
	a := &alternatingDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	g.SetRandomResults([]int{1, 0, 7})
	cost := DiscardRandomCost(1)
	for range 2 {
		if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err != nil {
			t.Fatal(err)
		}
	}
	if a.choices != 0 || len(a.Hand()) != 0 || len(a.Graveyard()) != 2 || a.Graveyard()[0].ID() != second.ID() || g.RandIntn(10) != 7 {
		t.Fatal("declaration retained payment state or used hand choice")
	}
}

func TestRandomDiscardCostFailedCommitClearsResults(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("Paid"))
	g.SetRandomResults([]int{0, 7})
	paid := BindNumber("paid life")
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCostInto(2, paid), DiscardRandomCost(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if tx.paidNumbers[paid] != 2 {
		t.Fatal("successful payment lost its result")
	}
	if err = tx.Commit(); err == nil {
		t.Fatal("stale total payment committed twice")
	}
	if len(tx.paidNumbers) != 0 || a.Life() != 18 || len(a.Graveyard()) != 1 || g.RandIntn(10) != 7 {
		t.Fatal("failed commit retained payment results or spent resources")
	}
}

func TestRandomDiscardCostZeroIgnoresHandAndRandomChanges(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewLand("Retained")
	a.AddToHand(card)
	g.SetRandomResults([]int{1, 7})
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardRandomCost(0)}})
	if err != nil {
		t.Fatal(err)
	}
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	g.RandIntn(10)
	if err = tx.Commit(); err != nil {
		t.Fatal("zero payment depends on an unrelated hand or RNG change", err)
	}
	if len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 || g.RandIntn(10) != 7 {
		t.Fatal("zero payment changed unrelated state")
	}
}
