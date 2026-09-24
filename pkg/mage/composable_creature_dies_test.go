package mage

import "testing"

func TestComposableCreatureDies(t *testing.T) {
	for _, scenario := range []string{"self", "other", "simultaneous", "noncreature", "exile", "phased", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1,
				WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if scenario == "noncreature" {
				other = NewArtifact("Other", "")
			}
			other.SetOwner(b.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			want := 21
			switch scenario {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.MutablePermanent(source.ID()).Damage = 1
				g.MutablePermanent(other.ID()).Damage = 1
				g.CheckStateBasedActions()
				want = 22
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
				want = 20
			default:
				if scenario == "phased" {
					g.PhaseOut(source.ID())
				}
				g.DestroyPermanent(g.FindPermanent(other.ID()))
				if scenario == "noncreature" || scenario == "phased" {
					want = 20
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != want-20 || a.Life() != 20 {
				t.Fatal("wrong pending death count or premature resolution")
			}
			if g.StackSize() > 0 && len(g.StackPeek().Targets) != 0 {
				t.Fatal("death participant became a target")
			}
			if scenario == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != want || a.Life() != 20 || g.StackSize() != 1 {
					t.Fatal("clone resolution changed the original")
				}
			}
			g.ResolveStack()
			if a.Life() != want {
				t.Fatalf("life = %d, want %d", a.Life(), want)
			}
		})
	}
}
