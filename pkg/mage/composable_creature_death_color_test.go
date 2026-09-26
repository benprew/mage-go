package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureDiesWithoutColorUsesDepartureState(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mana      string
		setColors []core.Color
		want      int
	}{
		{"red", "{R}", nil, 1},
		{"colorless", "", nil, 1},
		{"black", "{B}", nil, 0},
		{"black and red", "{B}{R}", nil, 0},
		{"became black", "{R}", []core.Color{core.Black}, 0},
		{"ceased to be black", "{B}", []core.Color{core.Blue}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithoutColor(core.Black), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			victim := NewCreature("Victim", tc.mana, 2, 2)
			victim.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.setColors != nil {
				if err := ApplyEffect(g, Until(core.EndOfTurn, SetColors(This, tc.setColors...)), victim.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack=%d, want %d", got, tc.want)
			}
			g.zones.ClearLKI()
			g.PutOnBattlefield(victim, b.PlayerID())
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life=%d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestCreatureDiesWithoutColorSimultaneousAndSourceState(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		watcher := NewCreature("Watcher", "{B}", 1, 1, WithAbility(WhenEvent(
			AnotherCreatureDies().WithoutColor(core.Black).Nontoken(), GainLifeFor(You, Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		red := NewCreature("Red", "{R}", 1, 1)
		red.SetOwner(b.PlayerID())
		black := NewCreature("Black", "{B}", 1, 1)
		black.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(red, b.PlayerID())
		g.PutOnBattlefield(black, b.PlayerID())
		if phase {
			g.PhaseOut(watcher.ID())
			g.DestroyPermanent(g.FindPermanent(red.ID()))
		} else {
			g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(red.ID()), g.FindPermanent(black.ID())})
		}
		g.PutTriggersOnStack()
		want := 1
		if phase {
			want = 0
		}
		if got := g.StackSize(); got != want {
			t.Fatalf("phase=%t stack=%d, want %d", phase, got, want)
		}
	}
}

func TestCreatureDiesWithoutColorRejectsInvalidColor(t *testing.T) {
	for _, color := range []core.Color{core.Colorless, core.AnyColor, core.Color(99)} {
		card := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
			CreatureDies().WithoutColor(color), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "color") {
			t.Fatalf("color=%v validation=%v", color, err)
		}
	}
	duplicate := NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithoutColor(core.Black).WithoutColor(core.Red), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "color") {
		t.Fatalf("duplicate filter validation=%v", err)
	}
}

func TestCreatureDiesWithoutColorIgnoresNonDeathsAndReturnedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().WithoutColor(core.Black), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "{R}", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.ExilePermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("exile matched death")
	}
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life=%d, want 21", a.Life())
	}
}
