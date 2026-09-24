package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageCleanupPhasedPermanentClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Phased survivor", "", 2, 3)
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, victim.ID())
	phased := g.PhaseOut(victim.ID())
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	if p := clone.FindPermanentIncludingPhased(victim.ID()); p == nil || !p.PhasedOut || p.Damage != 0 {
		t.Fatal("cleanup must remove damage and preserve phased-out status")
	}
	if p := g.FindPermanentIncludingPhased(victim.ID()); p == nil || !p.PhasedOut || p.Damage != 2 {
		t.Fatal("clone cleanup changed the original permanent")
	}
	clone.PhaseIn(phased)
	clone.DealDamageToPermanent(clone.FindPermanent(victim.ID()), 1, victim.ID())
	clone.checkStateBasedActions()
	if p := clone.FindPermanent(victim.ID()); p == nil || p.Damage != 1 {
		t.Fatal("damage from before cleanup must not kill the phased-in creature")
	}
	g.RunStepWithPriority(Cleanup)
	if g.FindPermanentIncludingPhased(victim.ID()).Damage != 0 || clone.FindPermanent(victim.ID()).Damage != 1 {
		t.Fatal("original cleanup did not remove damage independently")
	}
}

func TestDamageCleanupPhasedPermanentPriorityRounds(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Phased survivor", "", 2, 3)
	g.PutOnBattlefield(victim, b.PlayerID())
	for range 2 {
		g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, victim.ID())
		phased := g.PhaseOut(victim.ID())
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
		if !g.doCleanupActions() {
			t.Fatal("cleanup did not request priority")
		}
		if p := g.FindPermanentIncludingPhased(victim.ID()); p == nil || !p.PhasedOut || p.Damage != 0 {
			t.Fatal("each cleanup pass must remove damage before priority")
		}
		if !g.GetDamageSources(victim.ID())[victim.ID()] {
			t.Fatal("damage removal cleared source history before final cleanup")
		}
		g.ResolveStack()
		g.PhaseIn(phased)
	}
	g.RunStepWithPriority(Cleanup)
	if len(g.GetDamageSources(victim.ID())) != 0 || a.Life() != 22 {
		t.Fatal("final cleanup must clear history after both cleanup triggers resolve")
	}
}
