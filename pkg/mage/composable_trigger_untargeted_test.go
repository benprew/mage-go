package mage

import "testing"

func TestComposableEntryTriggerHasNoImplicitTarget(t *testing.T) {
	for _, scenario := range []string{"present", "departed", "phased", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			a.AddToLibrary(NewCreature("A draw", "", 1, 1))
			b.AddToLibrary(NewCreature("B draw", "", 1, 1))
			source := NewCreature("Entry source", "", 1, 1, WithAbility(WhenEvent(Enters(This), DrawFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutTriggersOnStack()
			obj := g.StackPeek()
			if obj == nil {
				t.Fatal("missing entry trigger")
			}
			if len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 {
				t.Fatal("untargeted entry trigger has an implicit target")
			}
			switch scenario {
			case "departed":
				g.ExilePermanent(g.FindPermanent(source.ID()))
			case "phased":
				g.PhaseOut(source.ID())
			case "copy":
				cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
				if cp == nil || len(cp.Targets) != 0 {
					t.Fatal("copy acquired a target")
				}
				g.ExilePermanent(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			if len(a.Hand()) != 1 {
				t.Fatal("source absence stopped the untargeted trigger")
			}
			want := 0
			if scenario == "copy" {
				want = 1
			}
			if len(b.Hand()) != want {
				t.Fatal("copy used the wrong controller")
			}
		})
	}
}

func TestComposableDeathTriggerHasNoImplicitTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToLibrary(NewCreature("Drawn", "", 1, 1))
	source := NewCreature("Death source", "", 1, 1, WithAbility(WhenEvent(Dies(This), DrawFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	obj := g.StackPeek()
	if obj == nil || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 {
		t.Fatal("death event participants became targets")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 {
		t.Fatal("death trigger did not draw")
	}
}
