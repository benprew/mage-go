package mage

import "testing"

func TestRegenerationShieldReturnedPermanent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.AddRegenerationShield(p.ID())
	g.RemoveFromBattlefield(p)
	p = g.PutOnBattlefield(card, a.PlayerID())
	if g.HasRegenerationShield(p.ID()) {
		t.Error("returned permanent inherited a shield")
	}
	g.DestroyPermanent(p)
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("old shield replaced destruction of a new permanent")
	}
}
