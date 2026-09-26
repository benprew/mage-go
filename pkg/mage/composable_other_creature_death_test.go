package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnotherCreatureYouControlDies(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"controlled", 1}, {"opponent", 0}, {"stolen", 1}, {"lost", 0},
		{"self", 0}, {"simultaneous", 1}, {"noncreature", 0},
		{"exile", 0}, {"token", 1}, {"phased", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlDies(), Sequence(GainLifeFor(You, Fixed(1)), AddCountersTo(This, core.P1P1, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if tc.name == "noncreature" {
				other = NewArtifact("Other", "")
			}
			if tc.name == "token" {
				other = NewToken("Other", 1, 1, []core.CardType{core.TypeCreature}, nil)
			}
			owner := a.PlayerID()
			if tc.name == "opponent" || tc.name == "stolen" {
				owner = b.PlayerID()
			}
			other.SetOwner(owner)
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, owner)
			if tc.name == "stolen" || tc.name == "lost" {
				controller := a.PlayerID()
				if tc.name == "lost" {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: other.ID(), ControllerID: controller, Duration: core.Indefinite})
			}
			switch tc.name {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
			default:
				if tc.name == "phased" {
					g.PhaseOut(source.ID())
				}
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || g.StackSize() != tc.want {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if a.Life() != 20+tc.want || clone.GetPlayer(a.PlayerID()).Life() != 20+tc.want {
				t.Fatal("wrong life result")
			}
			if perm := g.FindPermanent(source.ID()); perm != nil && tc.want == 1 && perm.Counters[core.P1P1] != 1 {
				t.Fatal("missing source counter")
			}
		})
	}
}

func TestAnotherCreatureYouControlDiesSourceIncarnation(t *testing.T) {
	for _, simultaneous := range []bool{false, true} {
		name := "after trigger"
		if simultaneous {
			name = "simultaneous"
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlDies(), Sequence(GainLifeFor(You, Fixed(1)), AddCountersTo(This, core.P1P1, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			other.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, a.PlayerID())
			if simultaneous {
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
			} else {
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing trigger")
			}
			if !simultaneous {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			g.PutOnBattlefield(source, b.PlayerID())
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("trigger controller changed")
			}
			if g.FindPermanent(source.ID()).Counters[core.P1P1] != 0 {
				t.Fatal("new incarnation received counter")
			}
		})
	}
}

func TestAnotherCreatureYouControlDiesCopyAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(source.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Text != "another creature you control dies" {
		t.Fatal("death filter missing from inspection")
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(other.ID()))
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || len(original.TargetSpecs) != 0 {
		t.Fatal("death event made a target")
	}
	if g.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("copy did not use its own controller")
	}
}
