package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageCleanupStateBasedDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Observer", "", 2, 2, WithAbility(
		CreatureDealtDamageBySourceDiesTrigger(GainLifeFor(You, Fixed(1)), false)))
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Temporary survivor", "", 1, 0)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(0), Fixed(3))), victim.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.RunStepWithPriority(Cleanup)
	if g.FindPermanent(victim.ID()) != nil || a.Life() != 21 {
		t.Fatalf("cleanup death must trigger once; life = %d, want 21", a.Life())
	}
	if g.GetDamageSources(victim.ID())[source.ID()] {
		t.Fatal("final cleanup retained damage history")
	}
}

func TestDamageCleanupPriorityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 2)
	victim := NewCreature("Victim", "", 2, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.SetDamageReflection(b.PlayerID(), victim.ID(), source.ID())
	register := func() {
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	}
	register()
	for round := range 2 {
		if !g.doCleanupActions() {
			t.Fatal("cleanup did not request priority")
		}
		if !g.GetDamageSources(victim.ID())[source.ID()] {
			t.Fatal("cleanup priority lost damage history")
		}
		if g.FindPermanent(victim.ID()).Damage != 0 {
			t.Fatal("cleanup retained marked damage")
		}
		if _, ok := g.damage.GetDamageReflection(b.PlayerID()); ok {
			t.Fatal("cleanup retained the expired reflection effect")
		}
		g.ResolveStack()
		if round == 0 {
			g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, victim.ID())
			register()
		}
	}
	if !g.GetDamageSources(victim.ID())[victim.ID()] {
		t.Fatal("second cleanup lost damage dealt during priority")
	}
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	if len(clone.GetDamageSources(victim.ID())) != 0 || len(g.GetDamageSources(victim.ID())) != 2 {
		t.Fatal("clone cleanup did not reset history independently")
	}
	if !clone.HasDealtDamageToPlayer(source.ID(), b.PlayerID()) || !clone.HasDealtDamageToPermanent(source.ID(), victim.ID()) {
		t.Fatal("final cleanup lost game damage history")
	}
	g.RunStepWithPriority(Cleanup)
	if len(g.GetDamageSources(victim.ID())) != 0 {
		t.Fatal("final cleanup retained damage history")
	}
}
