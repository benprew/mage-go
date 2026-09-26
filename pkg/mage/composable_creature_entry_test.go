package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnotherNontokenControlledCreatureEnters(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"controlled", 1}, {"opponent", 0}, {"stolen", 1}, {"lost", 0},
		{"self", 0}, {"token", 0}, {"noncreature", 0}, {"phased source", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().Nontoken(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			if tc.name == "self" {
				g.PutOnBattlefield(source, a.PlayerID())
			} else {
				g.PutOnBattlefield(source, a.PlayerID())
				other := NewCreature("Other", "", 1, 1)
				switch tc.name {
				case "token":
					other = NewToken("Other", 1, 1, []core.CardType{core.TypeCreature}, nil)
				case "noncreature":
					other = NewArtifact("Other", "")
				}
				owner, controller := a.PlayerID(), a.PlayerID()
				if tc.name == "opponent" || tc.name == "stolen" {
					owner = b.PlayerID()
				}
				if tc.name == "opponent" || tc.name == "lost" {
					controller = b.PlayerID()
				}
				other.SetOwner(owner)
				if tc.name == "phased source" {
					g.PhaseOut(source.ID())
				}
				g.PutOnBattlefield(other, controller)
			}
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack=%d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if a.Life() != 20+tc.want {
				t.Fatalf("life=%d, want %d", a.Life(), 20+tc.want)
			}
		})
	}
}

func TestAnotherNontokenControlledCreatureEntersInspection(t *testing.T) {
	card := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().Nontoken(), GainLifeFor(You, Fixed(1)))))
	meta := InspectDefinition(card.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Text != "another nontoken creature you control enters" {
		t.Fatalf("wrong inspection: %+v", meta)
	}
}

func TestAnotherNontokenControlledCreatureEntersSourceIdentityAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().Nontoken(), Sequence(
			GainLifeFor(You, Fixed(1)), AddCountersTo(This, core.P1P1, Fixed(1))))))
	source.SetOwner(a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("entry did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("stack copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("copy did not retain its own controller")
	}
	if got := g.FindPermanent(source.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("returned source got %d counters", got)
	}
}
