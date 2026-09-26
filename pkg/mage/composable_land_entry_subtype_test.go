package mage

import "testing"

func TestLandEntrySubtypeAtEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("entering controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().OfSubtype("Swamp").WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	wantLife := 20
	for _, tc := range []struct {
		subtype string
		want    bool
	}{
		{"Forest", false}, {"Swamp", true}, {"Mountain", false}, {"Swamp", true},
	} {
		land := NewLand("Land", WithSubTypes(tc.subtype))
		land.SetOwner(a.PlayerID())
		g.PutOnBattlefield(land, b.PlayerID())
		g.PutTriggersOnStack()
		want := 0
		if tc.want {
			want = 1
		}
		if g.StackSize() != want {
			t.Fatalf("%s: stack=%d, want %d", tc.subtype, g.StackSize(), want)
		}
		if tc.want {
			g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
			clone := g.Clone()
			clone.ResolveStack()
			if b.Life() != wantLife {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantLife++
		}
	}
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}

func TestLandEntrySubtypeDeclaration(t *testing.T) {
	for _, subtype := range []string{"", "Goblin"} {
		card := NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().OfSubtype(subtype), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted %q", subtype)
		}
	}
	card := NewArtifact("Duplicate", "", WithAbility(WhenEvent(LandEnters().OfSubtype("Swamp").OfSubtype("Forest"), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted duplicate subtype")
	}
}
