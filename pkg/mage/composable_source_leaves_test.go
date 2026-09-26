package mage

import "testing"

func TestSourceLeavesBattlefieldDestinationsAndIdentity(t *testing.T) {
	for _, destination := range []string{"graveyard", "exile", "hand"} {
		t.Run(destination, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Leaving watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			phased := g.PhaseOut(source.ID())
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phasing caused a leave trigger")
			}
			g.PhaseIn(phased)
			switch destination {
			case "graveyard":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "exile":
				g.ExilePermanent(g.FindPermanent(source.ID()))
			case "hand":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("stack=%d, want 1", g.StackSize())
			}
			old := g.StackPeek()
			if old == nil || len(old.TargetSpecs) != 0 {
				t.Fatal("leave trigger has targets")
			}
			g.PutOnBattlefield(source, b.PlayerID())
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("leave trigger changed controller after return")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared resolution state")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("cloned trigger did not resolve")
			}
		})
	}
}

func TestSourceLeavesBattlefieldSimultaneousAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(1)))))
	second := NewCreature("Second watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(1)))))
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatal("simultaneous leave or copied controller was lost")
	}
}

func TestSourceLeavesBattlefieldInspection(t *testing.T) {
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), DrawFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(source.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Text != "source leaves battlefield" {
		t.Fatal("leave pattern missing from inspection")
	}
	spell := NewSorcery("Invalid leave watcher", "", WithAbility(WhenEvent(LeavesBattlefield(This), DrawFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("spell accepted a battlefield source pattern")
	}
}
