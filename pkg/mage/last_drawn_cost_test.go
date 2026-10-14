package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLastDrawnCostIdentity(t *testing.T) {
	for _, state := range []string{"none", "drawn", "added", "returned", "later", "other-player", "reset", "empty-draw"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewLand("First"), NewLand("Second")
			first.SetOwner(a.PlayerID())
			second.SetOwner(a.PlayerID())
			a.AddToLibrary(first)
			cost := DiscardLastDrawnCardCost()
			want := state != "none" && state != "added" && state != "returned" && state != "reset"
			if state == "added" {
				a.AddToHand(first)
			} else if state != "none" {
				g.PlayerDrawCard(a)
			}
			selected := first
			switch state {
			case "returned":
				a.RemoveFromHand(first.ID())
				a.AddToHand(first)
			case "later":
				a.AddToLibrary(second)
				g.PlayerDrawCard(a)
				selected = second
			case "other-player":
				b.AddToLibrary(NewLand("Foreign"))
				g.PlayerDrawCard(b)
			case "reset":
				g.RunStepWithPriority(core.Cleanup)
			case "empty-draw":
				g.PlayerDrawCard(a)
			}
			before := len(a.Hand())
			for range 2 {
				if got := cost.CanPay(uuid.Nil, a.PlayerID(), g); got != want {
					t.Fatalf("CanPay = %v want %v", got, want)
				}
			}
			if len(a.Hand()) != before || len(a.Graveyard()) != 0 {
				t.Fatal("query moved cards")
			}
			err := cost.Pay(uuid.Nil, a.PlayerID(), g)
			if !want {
				if err == nil || len(a.Hand()) != before || len(a.Graveyard()) != 0 {
					t.Fatal("invalid payment changed zones")
				}
				return
			}
			if err != nil || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != selected.ID() || len(a.Hand()) != before-1 {
				t.Fatal("payment discarded the wrong card")
			}
			if cost.CanPay(uuid.Nil, a.PlayerID(), g) {
				t.Fatal("discarded visit can pay again")
			}
		})
	}
}

func TestLastDrawnCostTransaction(t *testing.T) {
	for _, change := range []string{"valid", "returned", "departed", "later", "reset", "duplicate", "unpaid"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			payment := NewLand("Payment")
			payment.SetOwner(a.PlayerID())
			a.AddToLibrary(payment)
			g.PlayerDrawCard(a)
			a.ManaPool().Add(core.Colorless, 2)
			mana := ManaCostOf("{2}")
			costs := []Cost{TapCost(This), DiscardLastDrawnCardCost()}
			if change == "duplicate" {
				costs = append(costs, DiscardLastDrawnCardCost())
			}
			if change == "unpaid" {
				mana = ManaCostOf("{3}")
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: append([]Cost{mana}, costs...)})
			if change != "duplicate" && change != "unpaid" {
				if err != nil {
					t.Fatal(err)
				}
				switch change {
				case "returned":
					a.RemoveFromHand(payment.ID())
					a.AddToHand(payment)
				case "departed":
					a.RemoveFromHand(payment.ID())
				case "later":
					a.AddToLibrary(NewLand("Later"))
					g.PlayerDrawCard(a)
				case "reset":
					a.ClearLastDrawnCard()
				}
				err = tx.Commit()
			}
			if change == "valid" {
				if err != nil || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped || len(a.Graveyard()) != 1 {
					t.Fatal("complete payment failed")
				}
			} else if err == nil || a.ManaPool().TotalMana() != 2 || g.FindPermanent(source.ID()).Tapped || len(a.Graveyard()) != 0 {
				t.Fatal("failed payment spent resources")
			}
		})
	}
}

func TestLastDrawnCostClone(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		t.Run(map[bool]string{false: "base", true: "wrapper"}[wrapped], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			var player Player = a
			if wrapped {
				player = &lastDrawnChoicePlayer{BasePlayer: a}
			}
			g := NewGame(player, b)
			a.AddToLibrary(NewLand("Payment"))
			g.PlayerDrawCard(player)
			cost := DiscardLastDrawnCardCost()
			clone := g.Clone()
			if err := cost.Pay(uuid.Nil, a.PlayerID(), clone); err != nil {
				t.Fatal(err)
			}
			if !cost.CanPay(uuid.Nil, a.PlayerID(), g) || cost.CanPay(uuid.Nil, a.PlayerID(), clone) {
				t.Fatal("clone shared history or hand")
			}
			if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type lastDrawnChoicePlayer struct{ *BasePlayer }

func (*lastDrawnChoicePlayer) ChooseCardsFromHand(int, string, GameReader) []Card {
	panic("exact payment asked for a choice")
}
func (*lastDrawnChoicePlayer) ChooseCardFromHand([]Card, string, GameReader) Card {
	panic("exact payment asked for a choice")
}

func TestLastDrawnCostDrawReplacement(t *testing.T) {
	for _, draws := range []bool{false, true} {
		t.Run(map[bool]string{false: "replaced", true: "generated"}[draws], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewLand("First"), NewLand("Second")
			a.AddToLibrary(first)
			a.AddToLibrary(second)
			g.PlayerDrawCard(a)
			var body = GainLifeFor(You, Fixed(1))
			if draws {
				body = DrawFor(You, Fixed(1))
			}
			if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, body), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PlayerDrawCard(a)
			cost := DiscardLastDrawnCardCost()
			if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err != nil {
				t.Fatal(err)
			}
			want := first.ID()
			if draws {
				want = second.ID()
			}
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != want {
				t.Fatal("replacement changed draw history incorrectly")
			}
		})
	}
}

func TestLastDrawnCostDiscardReplacement(t *testing.T) {
	for _, state := range []string{"library", "prevented", "recheck"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first := NewLand("Payment")
			first.SetOwner(a.PlayerID())
			a.AddToLibrary(first)
			g.PlayerDrawCard(a)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			if state == "prevented" {
				g.effects.AddReplacement(&preventOneCostDiscardReplacement{card: first.ID()})
			} else {
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
				if state == "recheck" {
					g.effects.AddReplacement(&discardToHandReplacement{})
				}
			}
			seen := 0
			watch := NewTriggered(core.EvtDiscard, false).SetCondition(func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
				if reader.GetPlayer(a.PlayerID()) != a {
					return false
				}
				seen++
				if len(reader.GetPlayer(a.PlayerID()).Hand()) != 0 || evt.SourceID != first.ID() || evt.DiscardedCardVisit != uuid.Nil {
					t.Error("replacement event lost payment facts")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watch)), b.PlayerID())
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{ManaCostOf("{2}"), TapCost(This), DiscardLastDrawnCardCost()}})
			if err == nil {
				err = tx.Commit()
			}
			if state == "library" {
				if err != nil || seen != 1 || len(a.Hand()) != 0 || len(a.Library()) != 1 || len(a.Graveyard()) != 0 || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("replaced cost was not paid")
				}
			} else if err == nil || seen != 0 || len(a.Hand()) != 1 || len(a.Library()) != 0 || a.ManaPool().TotalMana() != 2 || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("failed replacement spent resources")
			}
		})
	}
}

func TestLastDrawnCostInterleaved(t *testing.T) {
	a, b := &lastDrawnChoicePlayer{NewBasePlayer("A")}, &lastDrawnChoicePlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	first, second := NewLand("A payment"), NewLand("B payment")
	a.AddToLibrary(first)
	b.AddToLibrary(second)
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	cost := DiscardLastDrawnCardCost()
	one, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	two, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: b.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err = two.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = one.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.Graveyard()[0].ID() != first.ID() || b.Graveyard()[0].ID() != second.ID() {
		t.Fatal("payments shared history or choices")
	}
	if cost.CanPay(uuid.Nil, uuid.New(), g) || cost.Pay(uuid.Nil, uuid.New(), g) == nil {
		t.Fatal("absent payer accepted")
	}
	card := NewArtifact("Exact cost", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
