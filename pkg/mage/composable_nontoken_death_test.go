package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnotherNontokenCreatureDies(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"other controller", 1}, {"own controller", 1}, {"token", 0},
		{"self", 0}, {"simultaneous", 1}, {"phased source", 0}, {"absent source", 0},
		{"exile", 0}, {"returned source", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Observer", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureDies().Nontoken(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if tc.name == "token" {
				other = NewToken("Other", 1, 1, []core.CardType{core.TypeCreature}, nil)
			}
			other.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			controller := b.PlayerID()
			if tc.name == "own controller" {
				controller = a.PlayerID()
			}
			g.PutOnBattlefield(other, controller)
			if tc.name == "phased source" {
				g.PhaseOut(source.ID())
			}
			if tc.name == "absent source" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			switch tc.name {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
			default:
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			if tc.name == "returned source" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, a.PlayerID())
			}
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack = %d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life = %d, want %d", got, 20+tc.want)
			}
			if got := clone.GetPlayer(a.PlayerID()).Life(); got != 20+tc.want {
				t.Fatalf("clone life = %d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestAnotherNontokenCreatureDiesInspection(t *testing.T) {
	a := NewBasePlayer("A")
	source := NewCreature("Observer", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().Nontoken(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(source.Abilities()[0])
	if got := meta.Children[0].Text; got != "another nontoken creature dies" {
		t.Fatalf("pattern = %q", got)
	}
}

func TestAnotherNontokenCreatureDiesStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Observer", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().Nontoken(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(other.ID()))
	g.PutTriggersOnStack()
	if copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false); copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("life = %d/%d, want 21/21", a.Life(), b.Life())
	}
}
