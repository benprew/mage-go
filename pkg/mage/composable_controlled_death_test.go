package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureYouControlDies(t *testing.T) {
	for _, scenario := range []string{"own", "opponent", "stolen", "lost", "self", "simultaneous", "control after", "departed", "copy", "clone", "phased", "exile", "noncreature", "token"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureYouControlDies(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if scenario == "noncreature" {
				other = NewArtifact("Other", "")
			}
			if scenario == "token" {
				other = NewToken("Other", 1, 1, []core.CardType{core.TypeCreature}, nil)
			}
			owner := a.PlayerID()
			if scenario == "opponent" || scenario == "stolen" {
				owner = b.PlayerID()
			}
			other.SetOwner(owner)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, owner)
			if scenario == "stolen" || scenario == "lost" {
				controller := a.PlayerID()
				if scenario == "lost" {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: other.ID(), ControllerID: controller, Duration: core.Indefinite})
			}
			count := 1
			switch scenario {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
				count = 2
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
				count = 0
			default:
				if scenario == "phased" {
					g.PhaseOut(source.ID())
				}
				g.DestroyPermanent(g.FindPermanent(other.ID()))
				if scenario == "opponent" || scenario == "lost" || scenario == "phased" || scenario == "noncreature" {
					count = 0
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != count || a.Life() != 20 {
				t.Fatalf("stack=%d want=%d life=%d", g.StackSize(), count, a.Life())
			}
			switch scenario {
			case "control after":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "departed":
				g.ExilePermanent(g.FindPermanent(source.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
					t.Fatal("clone state is shared")
				}
			}
			g.ResolveStack()
			wantB := 20
			if scenario == "copy" {
				wantB++
			}
			if a.Life() != 20+count || b.Life() != wantB {
				t.Fatalf("life=%d/%d", a.Life(), b.Life())
			}
		})
	}
}
