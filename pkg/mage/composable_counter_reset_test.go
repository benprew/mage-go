package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterStateCondition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 2, 2)
	g.PutOnBattlefield(c, a.PlayerID())
	condition := SourceHasCounters(core.Tide, 4)
	ctx := StateContext{SourceID: c.ID(), ControllerID: a.PlayerID()}
	for _, n := range []int{0, 1, 3, 4, 5} {
		g.MutablePermanent(c.ID()).Counters[core.Tide] = uint8(n)
		for range 2 {
			if got := condition.checkState(g, ctx); got != (n >= 4) {
				t.Fatalf("count %d: condition = %v", n, got)
			}
		}
		if got := condition.checkState(g.Clone(), ctx); got != (n >= 4) {
			t.Fatal("clone condition differs")
		}
	}
	g.PhaseOut(c.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("phased source matched")
	}
	g.PhaseIn([]uuid.UUID{c.ID()})
	g.ExilePermanent(g.FindPermanent(c.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
}

func TestRemoveAllTypedCounters(t *testing.T) {
	for _, scenario := range []string{"present", "absent", "returned", "phased", "phase cycle"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", 2, 2)
			g.PutOnBattlefield(c, a.PlayerID())
			p := g.MutablePermanent(c.ID())
			p.AddCounter(core.Tide, 5)
			p.AddCounter(core.Charge, 2)
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "absent", "returned":
				g.ExilePermanent(g.FindPermanent(c.ID()))
				if scenario == "returned" {
					g.RemoveFromExile(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(c.ID()).AddCounter(core.Tide, 7)
				}
			case "phased":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			}
			clone := g.Clone()
			child := *ctx
			child.Game = clone
			if err := RemoveAllCounters(From(saved), core.Tide).Apply(&child); err != nil {
				t.Fatal(err)
			}
			if p := clone.FindPermanentIncludingPhased(c.ID()); p != nil {
				want := 0
				if scenario == "returned" {
					want = 7
				}
				if scenario == "phased" {
					want = 5
				}
				if int(p.Counters[core.Tide]) != want {
					t.Fatalf("tide = %d, want %d", p.Counters[core.Tide], want)
				}
				if scenario != "returned" && p.Counters[core.Charge] != 2 {
					t.Fatal("removed another counter kind")
				}
			}
			if p := g.FindPermanentIncludingPhased(c.ID()); p != nil && p.Counters[core.Tide] == 0 {
				t.Fatal("clone mutated original")
			}
		})
	}
}

func TestCounterResetValidation(t *testing.T) {
	for _, body := range []ResolvingEffect{RemoveAllCounters(nil, core.Tide), RemoveAllCounters(From(BindPermanent("missing")), core.Tide), RemoveAllCounters(DeclarePermanentTarget("missing", IsCreature), core.Tide)} {
		c := NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, body)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid subject accepted")
		}
	}
	c := NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(SourceHasCounters(core.Tide, -1), RemoveAllCounters(This, core.Tide))))
	if ValidateCardDefinition(c.Name(), c) == nil {
		t.Fatal("negative threshold accepted")
	}
}
