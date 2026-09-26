package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestHandVisitTracksZoneObject(t *testing.T) {
	p := NewBasePlayer("A")
	card := NewArtifact("Card", "")
	p.AddToHand(card)
	first := p.handVisit(card.ID())
	if first == uuid.Nil {
		t.Fatal("hand entry has no visit")
	}
	removed, ok := p.RemoveFromHand(card.ID())
	if !ok || p.handVisit(card.ID()) != uuid.Nil {
		t.Fatal("departed hand object remains live")
	}
	p.AddToHand(removed)
	if next := p.handVisit(card.ID()); next == first || next == uuid.Nil {
		t.Fatal("returned card reused the old hand visit")
	}
}

func TestHandVisitDrawReplacementAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Card", "")
	a.AddToLibrary(card)
	clone := g.Clone()
	cp := clone.GetPlayer(a.PlayerID()).(*SearchPlayer)
	if _, ok := clone.PlayerDrawCard(cp); !ok || cp.handVisit(card.ID()) == uuid.Nil {
		t.Fatal("clone draw has no hand visit")
	}
	if a.handVisit(card.ID()) != uuid.Nil {
		t.Fatal("clone draw changed original visits")
	}
	if _, ok := g.PlayerDrawCard(a); !ok || a.handVisit(card.ID()) == uuid.Nil {
		t.Fatal("draw has no hand visit")
	}
	first := a.handVisit(card.ID())
	a.SetHand([]Card{card})
	if a.handVisit(card.ID()) != first {
		t.Fatal("restoring an unchanged hand replaced its visit")
	}
	a.SetHand(nil)
	if a.handVisit(card.ID()) != uuid.Nil {
		t.Fatal("replaced hand kept old visit")
	}
	a.SetHand([]Card{card})
	if a.handVisit(card.ID()) == first || a.handVisit(card.ID()) == uuid.Nil {
		t.Fatal("new hand did not create a new visit")
	}
	clonedAgain := g.Clone()
	clonedPlayer := clonedAgain.GetPlayer(a.PlayerID()).(*SearchPlayer)
	if clonedPlayer.handVisit(card.ID()) != a.handVisit(card.ID()) {
		t.Fatal("clone lost the current hand visit")
	}
	if _, ok := clonedPlayer.RemoveFromHand(card.ID()); !ok {
		t.Fatal("clone could not remove the card")
	}
	if clonedPlayer.handVisit(card.ID()) != uuid.Nil || a.handVisit(card.ID()) == uuid.Nil {
		t.Fatal("clone removal changed original visit")
	}
}

func TestHandVisitAbsentAfterReplacedDrawAndDiscard(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Card", "")
	a.AddToLibrary(card)
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*Game, uuid.UUID) {})
	if _, ok := g.PlayerDrawCard(a); !ok {
		t.Fatal("first draw did not complete")
	}
	first := a.handVisit(card.ID())
	if first == uuid.Nil {
		t.Fatal("completed draw has no hand visit")
	}
	if _, ok := g.PlayerDrawCard(a); ok || a.handVisit(card.ID()) != first {
		t.Fatal("replaced draw changed hand identity")
	}
	if _, ok := a.DiscardCard(card.ID()); !ok || a.handVisit(card.ID()) != uuid.Nil {
		t.Fatal("discard retained the hand visit")
	}
}
