package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentColorTarget(t *testing.T) {
	for _, state := range []string{"legal", "multicolor", "colorless", "changed color", "phase", "phase cycle", "return", "shroud", "protection", "face down", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			filter := And(IsLand, Not(HasColorFilter(core.Black)))
			binding.spec.(*PermanentTarget).Filters = []PermanentFilter{filter}
			obj.Effects = []Effect{Destroy(binding)}
			g.ChangeObjectColor(p.ID(), core.Red)
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("complete color target rejected")
			}
			want := true
			switch state {
			case "multicolor":
				g.AddContinuousEffect(BecomesColors(p.ID(), []core.Color{core.Black, core.Red}, core.EndOfTurn))
				want = false
			case "colorless":
				g.AddContinuousEffect(BecomesColors(p.ID(), nil, core.EndOfTurn))
			case "changed color":
				g.ChangeObjectColor(p.ID(), core.Black)
				want = false
			case "phase", "phase cycle":
				g.PhaseOut(p.ID())
				want = state == "phase cycle"
				if want {
					g.PhaseIn([]uuid.UUID{p.ID()})
				}
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				want = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				want = false
			case "protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(core.Red))
				want = false
			case "face down":
				p.Card.SetBasePT(2, 2)
				p.FaceDown = true
				want = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "clone":
				clone := g.Clone()
				clone.ChangeObjectColor(p.ID(), core.Black)
				if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
					t.Fatal("clone ignored its current color")
				}
			}
			g.ApplyContinuousEffects()
			before := g.Clone()
			for range 2 {
				got, known := plan.Select(g, obj)
				if !known || (len(got) == 1) != want {
					t.Fatalf("selection=%v/%v want member=%v", got, known, want)
				}
				if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); !known || certain != want {
					t.Fatalf("certainty=%v/%v want=%v", certain, known, want)
				}
			}
			if !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.effects.replacements, before.effects.replacements) || g.StackSize() != before.StackSize() {
				t.Fatal("query changed game state")
			}
			g.ResolveStack()
			if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
				t.Fatal("query differs from native resolution")
			}
		})
	}
}

func TestPermanentColorTargetBoolean(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	g.ChangeObjectColor(p.ID(), core.Red)
	for _, tc := range []struct {
		filter PermanentFilter
		want   bool
	}{
		{HasColorFilter(core.Red), true},
		{HasColorFilter(core.Blue), false},
		{Or(HasColorFilter(core.Blue), And(IsLand, HasColorFilter(core.Red))), true},
		{Not(Or(HasColorFilter(core.Red), IsCreature)), false},
		{And(Or(HasColorFilter(core.Red), HasColorFilter(core.Blue)), Not(IsCreature)), true},
	} {
		tc.filter.fn = func(*Permanent, *Game) bool { panic("query ran typed callback") }
		binding.spec.(*PermanentTarget).Filters = []PermanentFilter{tc.filter}
		plan, ok := InspectPermanentTarget(binding)
		if !ok {
			t.Fatal("complete Boolean color tree rejected")
		}
		if got, known := plan.Select(g, obj); !known || (len(got) == 1) != tc.want {
			t.Fatal("Boolean color query differs")
		}
	}
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("query ran opaque callback") })
	for _, filter := range []PermanentFilter{
		Or(AnyPermanent, opaque), And(HasColorFilter(core.Red), opaque),
		Not(opaque), And(HasColorFilter(core.Red), AttackedThisTurn),
	} {
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("opaque", filter)); ok {
			t.Fatal("unsupported descendant supplied a partial query")
		}
	}
}
