package mage

import (
	"math"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFixedNumericTargetQueryBoundaries(t *testing.T) {
	for _, limit := range []int{math.MinInt, -2, 0, 2, math.MaxInt} {
		for _, value := range []int{math.MinInt, -3, -2, -1, 0, 1, 2, 3, math.MaxInt} {
			for _, tc := range []struct {
				filter PermanentFilter
				want   bool
			}{
				{HasPowerEqual(limit), value == limit},
				{HasToughnessEqual(limit), value == limit},
				{HasToughnessGTE(limit), value >= limit},
				{Or(Not(HasToughnessGTE(limit)), HasToughnessEqual(limit)), value <= limit},
				{And(HasPowerEqual(limit), HasToughnessEqual(limit)), value == limit},
				{Or(And(IsCreature, HasPowerEqual(limit)), Not(HasToughnessGTE(limit))), value == limit || value < limit},
			} {
				g, a, _ := newSpellCopyGame()
				p := g.PutOnBattlefield(NewCreature("Subject", "", value, value), a.PlayerID())
				binding := DeclarePermanentTarget("numeric", tc.filter)
				card := NewSorcery("Source", "")
				obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: []Target{binding.targetSpec()}}
				g.PushStack(obj)
				native := slices.Contains(binding.Possible(a.PlayerID(), card, g), p.ID())
				if native != tc.want {
					t.Fatalf("native %s for %d: %v", tc.filter.Text(), value, native)
				}
				plan, ok := InspectPermanentTarget(binding)
				if !ok {
					t.Fatalf("opaque numeric filter: %s", tc.filter.Text())
				}
				binding.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("query called filter") }
				got, known := plan.Select(g, obj)
				if !known || (len(got) == 1) != tc.want {
					t.Fatalf("%s for %d: %v/%v", tc.filter.Text(), value, got, known)
				}
			}
		}
	}
}

func TestFixedNumericTargetQueryIdentity(t *testing.T) {
	for _, state := range []string{"legal", "counter", "depart", "return", "phase", "phase cycle", "source leaves", "copy", "clone", "shroud"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			binding := DeclarePermanentTarget("subject", And(IsCreature, HasPowerEqual(1), HasToughnessEqual(2)))
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), Destroy(binding), WithTargets(binding)))), a.PlayerID())
			card := NewCreature("Subject", "", 1, 2)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("numeric target opaque")
			}
			want := true
			switch state {
			case "counter":
				p.AddCounter(core.P1P1, 1)
				want = false
			case "depart":
				g.RemoveFromBattlefield(p)
				want = false
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, b.PlayerID())
				want = false
			case "phase":
				g.PhaseOut(p.ID())
				want = false
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "source leaves":
				g.RemoveFromBattlefield(source)
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			case "clone":
				clone := g.Clone()
				clone.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
				if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
					t.Fatal("clone ignored current values")
				}
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				want = false
			}
			g.ApplyContinuousEffects()
			got, known := plan.Select(g, obj)
			if !known || (len(got) == 1) != want {
				t.Fatalf("selection %v/%v want %v", got, known, want)
			}
			g.ResolveStack()
			if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want && state != "depart" {
				t.Fatal("resolution differs from inspection")
			}
		})
	}
}

func TestFixedNumericTargetQueryOpaque(t *testing.T) {
	trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("query called opaque filter") })
	for _, f := range []PermanentFilter{And(HasPowerEqual(1), trap), Or(HasToughnessGTE(2), trap), Not(And(HasToughnessEqual(1), trap))} {
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("opaque", f)); ok {
			t.Fatal("partial numeric tree accepted")
		}
	}
}

func TestFixedNumericTargetQueryLayers(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 3), a.PlayerID())
	card := NewSorcery("Source", "")
	binding := DeclarePermanentTarget("subject", And(HasPowerEqual(9), HasToughnessEqual(3)))
	obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: []Target{binding.targetSpec()}}
	g.PushStack(obj)
	plan, ok := InspectPermanentTarget(binding)
	if !ok {
		t.Fatal("numeric target opaque")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetBasePowerToughness(This, Fixed(2), Fixed(7)), ModifyPT(This, Fixed(0), Fixed(1)), SwitchPowerToughness(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
	g.ApplyContinuousEffects()
	if got, known := plan.Select(g, obj); !known || len(got) != 1 {
		t.Fatal("query ignored final layer values", got, known)
	}
	if !slices.Contains(binding.Possible(a.PlayerID(), card, g), p.ID()) {
		t.Fatal("native layer values differ")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if got, known := plan.Select(g, obj); !known || len(got) != 0 {
		t.Fatal("query retained expired layer values")
	}
}

func TestFixedNumericTargetQueryCollections(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 2), a.PlayerID())
	filter := And(IsCreature, HasPowerEqual(1), HasToughnessGTE(2))
	filter.fn = func(*Permanent, *Game) bool { panic("collection query called filter") }
	plan, ok := InspectPermanentCollection(MatchingPermanents(filter))
	if !ok {
		t.Fatal("shared numeric collection opaque")
	}
	if got := plan.Select(g); len(got) != 1 || got[0].Incarnation != p.incarnationID {
		t.Fatal("numeric collection lost visit")
	}
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
	if len(plan.Select(g)) != 0 {
		t.Fatal("numeric collection ignored current power")
	}
}
