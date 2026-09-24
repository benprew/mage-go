package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestZoneLKICleanupPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Departed source", "{3}", 2, 2)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(source.ID()))
	register := func() {
		g.RegisterDelayedTrigger(&DelayedTrigger{
			EventType: EvtCleanup, TargetID: source.ID(), Controller: a.PlayerID(),
			Effects: []Effect{GainLifeTarget(TargetPermanentManaValue())},
		})
	}
	register()
	for round := range 2 {
		g.zones.AddExileInstead(source.ID(), source.ID())
		if !g.doCleanupActions() {
			t.Fatal("cleanup did not request priority")
		}
		if g.LKI(source.ID()) == nil {
			t.Fatal("cleanup priority lost departure information")
		}
		if g.zones.IsExileInstead(source.ID()) {
			t.Fatal("cleanup priority retained an expired exile effect")
		}
		if round == 0 {
			clone := g.Clone()
			clone.ResolveStack()
			clone.RunStepWithPriority(Cleanup)
			if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 || g.LKI(source.ID()) == nil {
				t.Fatal("cloned cleanup trigger lost LKI or changed the original game")
			}
		}
		g.ResolveStack()
		if a.Life() != 20+3*(round+1) {
			t.Fatalf("cleanup trigger life = %d", a.Life())
		}
		if round == 0 {
			register()
		}
	}
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	if clone.LKI(source.ID()) != nil || g.LKI(source.ID()) == nil {
		t.Fatal("clone cleanup did not clear its cache independently")
	}
	g.RunStepWithPriority(Cleanup)
	if g.LKI(source.ID()) != nil {
		t.Fatal("final cleanup retained the legacy cache")
	}
}

func TestZoneLKICleanupStateBasedDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Temporary survivor", "{4}", 1, 0)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(0), Fixed(1))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	added := false
	g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
		if game.LKI(source.ID()) == nil {
			t.Fatal("cleanup death lost LKI before priority ended")
		}
		if !added {
			added = true
			game.RegisterDelayedTrigger(&DelayedTrigger{
				EventType: EvtCleanup, TargetID: source.ID(), Controller: a.PlayerID(),
				Effects: []Effect{GainLifeTarget(TargetPermanentManaValue())},
			})
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Cleanup)
	if !added || a.Life() != 24 || g.GetCleanupPriorityRounds() != 2 {
		t.Fatalf("priority = %v, life = %d, rounds = %d", added, a.Life(), g.GetCleanupPriorityRounds())
	}
	if g.LKI(source.ID()) != nil {
		t.Fatal("final cleanup retained the cleanup death cache")
	}
}
