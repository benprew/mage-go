package mage

import (
	"fmt"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type alternatingDiscardPlayer struct {
	*BasePlayer
	choices int
}

type invalidDiscardPlayer struct {
	*BasePlayer
	choice Card
}

func (p *invalidDiscardPlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	return []Card{p.choice}
}

func (p *alternatingDiscardPlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	p.choices++
	hand := p.Hand()
	return []Card{hand[(len(hand)-p.choices+1)%len(hand)]}
}

func TestDiscardPaymentKeepsOneChosenCard(t *testing.T) {
	a := &alternatingDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.choices != 1 || len(a.Hand()) != 1 || a.Hand()[0].ID() != second.ID() {
		t.Fatalf("discard cost used another choice: choices=%d hand=%v", a.choices, a.Hand())
	}
}

func TestDiscardPaymentRejectsMissingChoice(t *testing.T) {
	a := &alternatingDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(1)}})
	if err != nil {
		t.Fatal(err)
	}
	a.RemoveFromHand(first.ID())
	if err := tx.Commit(); err == nil {
		t.Fatal("missing chosen card did not fail payment")
	}
	if a.choices != 1 || len(a.Hand()) != 1 || a.Hand()[0].ID() != second.ID() {
		t.Fatal("failed payment discarded a different card")
	}
}

func TestDiscardPaymentRejectsInvalidChoiceBeforeCommit(t *testing.T) {
	a := &invalidDiscardPlayer{BasePlayer: NewBasePlayer("A"), choice: NewLand("Outside")}
	g := NewGame(a, NewBasePlayer("B"))
	card := NewLand("In hand")
	a.AddToHand(card)
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(1)}}); err == nil {
		t.Fatal("choice outside the hand was accepted")
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() {
		t.Fatal("invalid payment changed the hand")
	}
}

func TestDiscardPaymentDeclarationCanBeReused(t *testing.T) {
	a := &alternatingDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	cost := DiscardCost(1)
	for range 2 {
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.Hand()) != 0 || a.choices != 2 {
		t.Fatal("cost declaration kept a previous payment choice")
	}
}

func TestDiscardPaymentRejectsReturnedHandVisit(t *testing.T) {
	for _, count := range []int{1, 2} {
		for _, capture := range []bool{false, true} {
			if capture && count != 1 {
				continue
			}
			t.Run(fmt.Sprintf("count=%d/capture=%t", count, capture), func(t *testing.T) {
				a := NewBasePlayer("A")
				g := NewGame(a, NewBasePlayer("B"))
				for range count {
					a.AddToHand(NewLand("Chosen"))
				}
				a.ManaPool().Add(Red, 1)
				paid := BindCard("paid")
				cost := DiscardCost(count)
				if capture {
					cost = DiscardCardCostInto(paid)
				}
				tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
					Controller: a.PlayerID(), Costs: []Cost{ManaPayment("{R}"), LifePayCost(2), cost},
				})
				if err != nil {
					t.Fatal(err)
				}
				card, _ := a.RemoveFromHand(a.Hand()[count-1].ID())
				a.AddToHand(card)
				if err := tx.Commit(); err == nil {
					t.Fatal("returned card paid a locked discard cost")
				}
				if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Hand()) != count || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 || len(tx.paidCards) != 0 {
					t.Fatal("stale choice spent resources or published a result")
				}
			})
		}
	}
}

func TestDiscardPaymentHandVisitsRemainLocal(t *testing.T) {
	a := &alternatingDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("First"))
	a.AddToHand(NewLand("Second"))
	cost := DiscardCost(1)
	first, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clonedPayment := *first
	clonedPayment.game = clone
	card, _ := a.RemoveFromHand(a.Hand()[0].ID())
	a.AddToHand(card)
	if err := first.Commit(); err == nil {
		t.Fatal("returned live card paid the first transaction")
	}
	if err := clonedPayment.Commit(); err != nil {
		t.Fatalf("live departure changed the cloned visit: %v", err)
	}
	if len(a.Hand()) != 2 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("clone payment changed the live hand")
	}
	if err := second.Commit(); err != nil {
		t.Fatalf("another transaction changed the second choice: %v", err)
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() || a.choices != 2 {
		t.Fatal("commit chose again or discarded the wrong card")
	}
}

func TestDiscardPaymentAcceptsUnchangedHandVisits(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			for range count {
				a.AddToHand(NewLand("Chosen"))
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(count)}})
			if err != nil {
				t.Fatal(err)
			}
			a.SetHand(append([]Card(nil), a.Hand()...))
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 0 || len(a.Graveyard()) != count || g.PlayerDiscardCountThisTurn(a.PlayerID()) != count {
				t.Fatal("unchanged visits did not pay the full discard")
			}
		})
	}
}

type fixedDiscardChoicesPlayer struct {
	*BasePlayer
	selected []Card
}

func (p *fixedDiscardChoicesPlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	return p.selected
}

func TestDiscardPaymentRejectsInvalidMultiCardChoices(t *testing.T) {
	for _, choice := range []string{"duplicate", "too few", "outside", "nil"} {
		t.Run(choice, func(t *testing.T) {
			a := &fixedDiscardChoicesPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			switch choice {
			case "duplicate":
				a.selected = []Card{first, first}
			case "too few":
				a.selected = []Card{first}
			case "outside":
				a.selected = []Card{first, NewLand("Outside")}
			case "nil":
				a.selected = []Card{first, nil}
			}
			if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), DiscardCost(2)}}); err == nil {
				t.Fatal("invalid choices paid the total cost")
			}
			if a.Life() != 20 || len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
				t.Fatal("invalid preparation changed live state")
			}
		})
	}
}

func TestDiscardPaymentSyntheticTwoCardActivation(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Discard contract", "", WithAbility(Activate(
		Costs(DiscardCost(2)), GainLifeFor(You, Fixed(2)),
	)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToHand(NewLand("First"))
	a.AddToHand(NewLand("Second"))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	g.Clone().ResolveStack()
	if a.Life() != 20 || len(a.Graveyard()) != 2 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 2 {
		t.Fatal("copy or clone paid the cost again")
	}
	g.ResolveStack()
	if a.Life() != 24 || len(a.Graveyard()) != 2 {
		t.Fatal("paid activation and its copy did not resolve independently")
	}
}
