package mage

import (
	"testing"
)

func TestChangeOwnerPreservesCardAliases(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	card := NewCreature("Card", "{0}", 1, 1)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.stack.Push(&StackObject{Card: card, TargetSource: card, CastContext: &CastContext{RevealedAtCast: []*Card{card}}})
	if err := g.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
		t.Fatal(err)
	}
	obj := g.stack.Peek()
	if p.Card != card || obj.Card.Owner() != b.PlayerID() || obj.TargetSource.Owner() != b.PlayerID() || obj.CastContext.RevealedAtCast[0].Owner() != b.PlayerID() {
		t.Fatal("ownership change broke aliases")
	}
}
