package mage

import "testing"

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
