package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func ownedTestCard(owner uuid.UUID, name string) Card {
	c := NewArtifact(name, "{1}")
	c.SetOwner(owner)
	return c
}

func TestNewGameWithAnte(t *testing.T) {
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	a1 := ownedTestCard(a.PlayerID(), "A1")
	a2 := ownedTestCard(a.PlayerID(), "A2")
	b1 := ownedTestCard(b.PlayerID(), "B1")
	a.SetLibrary([]Card{a1, a2})
	b.SetLibrary([]Card{b1})

	g, err := NewGameWithAnte(a, b, []Card{a1, a2}, []Card{b1})
	if err != nil {
		t.Fatalf("NewGameWithAnte: %v", err)
	}
	if !g.AnteEnabled() || len(a.Library()) != 0 || len(b.Library()) != 0 {
		t.Fatalf("ante game was not initialized: enabled=%v libraries=%d/%d", g.AnteEnabled(), len(a.Library()), len(b.Library()))
	}
	cards, err := g.AnteCards()
	if err != nil || len(cards) != 3 {
		t.Fatalf("AnteCards = %d, %v", len(cards), err)
	}
	owned, err := g.AnteCardsOwnedBy(a.PlayerID())
	if err != nil || len(owned) != 2 {
		t.Fatalf("AnteCardsOwnedBy(A) = %d, %v", len(owned), err)
	}
}

func TestNewGameWithAnteValidation(t *testing.T) {
	tests := []struct {
		name  string
		setup func(a, b *BasePlayer) ([]Card, []Card)
	}{
		{"wrong owner", func(a, b *BasePlayer) ([]Card, []Card) {
			c := ownedTestCard(b.PlayerID(), "x")
			a.SetLibrary([]Card{c})
			return []Card{c}, nil
		}},
		{"duplicate", func(a, b *BasePlayer) ([]Card, []Card) {
			c := ownedTestCard(a.PlayerID(), "x")
			a.SetLibrary([]Card{c})
			return []Card{c, c}, nil
		}},
		{"missing", func(a, b *BasePlayer) ([]Card, []Card) { c := ownedTestCard(a.PlayerID(), "x"); return []Card{c}, nil }},
		{"token", func(a, b *BasePlayer) ([]Card, []Card) {
			c := NewToken("x", 1, 1, []CardType{TypeCreature}, nil)
			c.SetOwner(a.PlayerID())
			a.SetLibrary([]Card{c})
			return []Card{c}, nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			aa, bb := tt.setup(a, b)
			if _, err := NewGameWithAnte(a, b, aa, bb); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestAnteOperationsRequireAnteAndOwner(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	c := ownedTestCard(a.PlayerID(), "card")
	a.SetLibrary([]Card{c})
	plain := NewGame(a, b)
	if _, err := plain.AnteCards(); err == nil {
		t.Fatal("AnteCards succeeded with ante disabled")
	}
	if err := plain.MoveToAnte(a.PlayerID(), c.ID()); err == nil {
		t.Fatal("MoveToAnte succeeded with ante disabled")
	}
	if err := plain.ChangeOwner(c.ID(), b.PlayerID()); err == nil {
		t.Fatal("ChangeOwner succeeded with ante disabled")
	}
	if _, err := plain.AnteResult(); err == nil {
		t.Fatal("AnteResult succeeded with ante disabled")
	}

	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.MoveToAnte(b.PlayerID(), c.ID()); err == nil {
		t.Fatal("opponent anted a card they do not own")
	}
	if err := g.MoveToAnte(a.PlayerID(), c.ID()); err != nil {
		t.Fatalf("MoveToAnte: %v", err)
	}
	if g.CardZone(c.ID()) != ZoneAnte || len(a.Library()) != 0 {
		t.Fatal("card did not move from library to ante")
	}
	removed, err := g.RemoveFromAnte(c.ID(), ZoneHand)
	if err != nil || removed.ID() != c.ID() || g.CardZone(c.ID()) != ZoneAny {
		t.Fatalf("RemoveFromAnte = %v, %v, zone %v", removed, err, g.CardZone(c.ID()))
	}
}

func TestChangeOwnerPreservesZoneControllerAndSupportsReversal(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := ownedTestCard(a.PlayerID(), "tablet target")
	perm := g.PutOnBattlefield(c, b.PlayerID())
	if err := g.ChangeOwner(c.ID(), b.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(c.ID()); got == nil || got.Card.Owner() != b.PlayerID() || got.ControllerID() != b.PlayerID() {
		t.Fatal("ownership change altered zone/controller or did not update owner")
	}
	if perm.ID() != c.ID() || g.CardZone(c.ID()) != ZoneBattlefield {
		t.Fatal("ownership change altered identity or zone")
	}
	if err := g.ChangeOwner(c.ID(), a.PlayerID()); err != nil {
		t.Fatal(err)
	}
	a.SetLife(0)
	result, err := g.AnteResult()
	if err != nil || len(result) != 0 {
		t.Fatalf("reversed transfer result = %#v, %v", result, err)
	}
}

func TestChangeOwnerValidationAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	c := ownedTestCard(a.PlayerID(), "card")
	a.SetLibrary([]Card{c})
	g, _ := NewGameWithAnte(a, b, nil, nil)
	if err := g.ChangeOwner(uuid.New(), b.PlayerID()); err == nil {
		t.Fatal("missing card accepted")
	}
	if err := g.ChangeOwner(c.ID(), uuid.New()); err == nil {
		t.Fatal("missing player accepted")
	}
	clone := g.Clone()
	if err := clone.ChangeOwner(c.ID(), b.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if clone.FindCardAnywhere(c.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(c.ID()).Owner() != a.PlayerID() {
		t.Fatal("clone ownership mutation leaked to parent")
	}
}

func TestChangeOwnerReplacesCardsInEveryModeledZone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)

	library := ownedTestCard(a.PlayerID(), "library")
	hand := ownedTestCard(a.PlayerID(), "hand")
	graveyard := ownedTestCard(a.PlayerID(), "graveyard")
	ante := ownedTestCard(a.PlayerID(), "ante")
	exile := ownedTestCard(a.PlayerID(), "exile")
	battlefield := ownedTestCard(a.PlayerID(), "battlefield")
	stack := ownedTestCard(a.PlayerID(), "stack")
	a.SetLibrary([]Card{library})
	a.SetHand([]Card{hand})
	a.SetGraveyard([]Card{graveyard})
	a.SetAnte([]Card{ante})
	g.ExileCard(exile, uuid.Nil)
	perm := g.PutOnBattlefield(battlefield, b.PlayerID())
	g.stack.Push(&StackObject{ID: uuid.New(), Card: stack, Controller: a.PlayerID()})

	for _, card := range []Card{library, hand, graveyard, ante, exile, battlefield, stack} {
		if err := g.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
			t.Fatalf("ChangeOwner(%s): %v", card.Name(), err)
		}
		if got := g.FindCardAnywhere(card.ID()); got == nil || got.Owner() != b.PlayerID() || got.ID() != card.ID() {
			t.Fatalf("%s owner/identity not replaced", card.Name())
		}
	}
	if g.FindPermanent(perm.ID()).ControllerID() != b.PlayerID() || g.stack.Peek().Controller != a.PlayerID() {
		t.Fatal("ownership replacement changed a controller")
	}
}

func TestAnteResultSettlesAndIsIdempotent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	a1 := ownedTestCard(a.PlayerID(), "A ante")
	b1 := ownedTestCard(b.PlayerID(), "B ante")
	a.SetLibrary([]Card{a1})
	b.SetLibrary([]Card{b1})
	g, _ := NewGameWithAnte(a, b, []Card{a1}, []Card{b1})
	transferred := ownedTestCard(b.PlayerID(), "other")
	b.AddToHand(transferred)
	if err := g.ChangeOwner(transferred.ID(), a.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if _, err := g.AnteResult(); err == nil {
		t.Fatal("unfinished game settled")
	}
	b.SetLife(0)
	got, err := g.AnteResult()
	if err != nil || len(got) != 2 {
		t.Fatalf("AnteResult = %#v, %v", got, err)
	}
	if g.FindCardAnywhere(b1.ID()).Owner() != a.PlayerID() {
		t.Fatal("winner did not receive opponent ante card")
	}
	got[0].CardName = "mutated"
	again, err := g.AnteResult()
	if err != nil || len(again) != 2 || again[0].CardName == "mutated" {
		t.Fatal("AnteResult was not idempotent/defensive")
	}
}

func TestAnteResultRejectsWinnerlessGame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)
	a.SetLife(0)
	b.SetLife(0)
	if _, err := g.AnteResult(); err == nil {
		t.Fatal("winnerless game settled")
	}
}
