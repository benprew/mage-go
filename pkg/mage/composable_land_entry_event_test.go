package mage

import "testing"

func TestLandEntryControllerEvent(t *testing.T) {
	for _, tc := range []struct {
		name        string
		land        bool
		phaseSource bool
		want        int
	}{
		{"own land", true, false, 1},
		{"opponent land", true, false, 1},
		{"nonland", false, false, 0},
		{"phased source", true, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			controller := BindPlayer("entry controller")
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				LandEnters().WithController(controller), GainLifeFor(FromPlayer(controller), Fixed(1)))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phaseSource {
				g.PhaseOut(watcher.ID())
			}
			var card Card = NewArtifact("Other", "")
			if tc.land {
				card = NewLand("Other")
			}
			controllerID := b.PlayerID()
			if tc.name == "own land" {
				controllerID = a.PlayerID()
			}
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, controllerID)
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantA, wantB := 20, 20
			if tc.want == 1 {
				if controllerID == a.PlayerID() {
					wantA++
				} else {
					wantB++
				}
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=%d,%d want=%d,%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestLandEntryControllerDeclaration(t *testing.T) {
	binding := BindPlayer("controller")
	good := NewArtifact("Good", "", WithAbility(WhenEvent(LandEnters().WithController(binding),
		GainLifeFor(FromPlayer(binding), Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(good.Abilities()[0]); got.Role != "event-trigger" || len(got.Children) == 0 {
		t.Fatalf("inspection: %+v", got)
	}
	for _, pattern := range []landEntersPattern{LandEnters().WithController(nil), LandEnters().WithController(BindPlayer("")), LandEnters().WithController(binding).WithController(binding)} {
		bad := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid binding")
		}
	}
	unbound := NewArtifact("Unbound", "", WithAbility(WhenEvent(LandEnters(), GainLifeFor(FromPlayer(binding), Fixed(1)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound read")
	}
}

func TestLandEntryControllerPendingCopyAndReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().WithController(controller), DealDamageFrom(This, FromPlayer(controller), Fixed(2)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land")
	land.SetOwner(a.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("land did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 16 {
		t.Fatalf("entry controller capture lost: %d,%d", a.Life(), b.Life())
	}
}
