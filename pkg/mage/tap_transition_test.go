package mage

import (
	"testing"
)

func TestTapPermanentRequiresTransition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	aura := NewAura("Watcher", "", WithCastTarget(TargetLand()), WithAbility(WhenAttachedBecomesTappedTrigger(GainLifeFor(You, Fixed(1)), false)))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), land.ID())
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("one tap transition queued %d triggers", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life = %d", a.Life())
	}
	g.MutablePermanent(land.ID()).Tapped = false
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("second transition did not trigger")
	}
}
