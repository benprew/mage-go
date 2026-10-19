package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSubtypeTarget(t *testing.T) {
	for _, state := range []string{"legal", "changed subtype", "phase", "phase cycle", "return", "shroud", "protection", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			if err := ApplyEffect(g, Until(core.EndOfTurn, SetSubtypeFamily(This, SubtypeLand, "Forest")), p.ID(), p.ControllerID(), nil); err != nil {
				t.Fatal(err)
			}
			spec := binding.spec.(*PermanentTarget)
			spec.Filters = []PermanentFilter{And(IsLand, Not(HasSubType("Island")))}
			spec.requiredSubtypes = []string{"Forest"}
			spec.subtypeDomains = []PermanentTargetDomain{AllPermanentDomains(SubtypeDomain("Forest"), AnyPermanentDomain(NonSubtypeDomain("Island"), SubtypeDomain("Mountain")))}
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("complete subtype target rejected")
			}
			obj.Effects = []Effect{Destroy(binding)}
			want := true
			switch state {
			case "changed subtype":
				if err := ApplyEffect(g, Until(core.EndOfTurn, SetSubtypeFamily(This, SubtypeLand, "Island")), p.ID(), p.ControllerID(), nil); err != nil {
					t.Fatal(err)
				}
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
				if err := ApplyEffect(g, Until(core.EndOfTurn, GrantKeywordModification(This, core.Shroud)), p.ID(), p.ControllerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = false
			case "protection":
				if err := ApplyEffect(g, Until(core.EndOfTurn, GrantProtectionColors(This, core.Red)), p.ID(), p.ControllerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "clone":
				clone := g.Clone()
				if err := ApplyEffect(clone, Until(core.EndOfTurn, SetSubtypeFamily(This, SubtypeLand, "Island")), p.ID(), p.ControllerID(), nil); err != nil {
					t.Fatal(err)
				}
				if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
					t.Fatal("clone reused original subtype")
				}
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()) == nil {
					t.Fatal("clone resolved an illegal target")
				}
			}
			g.ApplyContinuousEffects()
			for range 2 {
				got, known := plan.Select(g, obj)
				if !known || (len(got) == 1) != want {
					t.Fatalf("selection=%v/%v want=%v", got, known, want)
				}
				if len(got) > 0 && got[0] != (PermanentVisit{p.ID(), p.incarnationID}) {
					t.Fatal("wrong visit")
				}
				if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); !known || certain != want {
					t.Fatalf("destruction=%v/%v want=%v", certain, known, want)
				}
			}
			g.ResolveStack()
			if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
				t.Fatal("query differs from resolution")
			}
		})
	}
}

func TestPermanentSubtypeTargetDomains(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetSubtypeFamily(This, SubtypeLand, "Forest")), p.ID(), p.ControllerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		domain PermanentTargetDomain
		want   bool
	}{
		{SubtypeDomain("Forest"), true}, {NonSubtypeDomain("Forest"), false},
		{AnyPermanentDomain(SubtypeDomain("Island"), SubtypeDomain("Forest")), true},
		{AllPermanentDomains(SubtypeDomain("Forest"), NonSubtypeDomain("Forest")), false},
		{AnyPermanentDomain(UnknownPermanentDomain(), SubtypeDomain("Island")), true},
	} {
		spec := binding.spec.(*PermanentTarget)
		spec.subtypeDomains = []PermanentTargetDomain{tc.domain}
		spec.Filters = []PermanentFilter{Or(HasSubType("Forest"), HasColorFilter(core.Red))}
		spec.Filters[0].fn = func(*Permanent, *Game) bool { panic("query ran filter") }
		plan, ok := InspectPermanentTarget(binding)
		if !ok {
			t.Fatal("complete domain rejected")
		}
		if got, known := plan.Select(g, obj); !known || (len(got) == 1) != tc.want {
			t.Fatal("domain ignored")
		}
	}
	for _, domain := range []PermanentTargetDomain{{}, SubtypeDomain(""), NonSubtypeDomain(" Forest"), AllPermanentDomains(), AnyPermanentDomain(SubtypeDomain("Forest"), PermanentTargetDomain{})} {
		binding.spec.(*PermanentTarget).subtypeDomains = []PermanentTargetDomain{domain}
		if _, ok := InspectPermanentTarget(binding); ok {
			t.Fatal("invalid domain accepted")
		}
	}
	binding.spec.(*PermanentTarget).subtypeDomains = nil
	binding.spec.(*PermanentTarget).requiredSubtypes = []string{""}
	if _, ok := InspectPermanentTarget(binding); ok {
		t.Fatal("invalid required subtype accepted")
	}
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque query") })
	if _, ok := InspectPermanentTarget(DeclarePermanentTarget("partial", Or(HasSubType("Forest"), opaque))); ok {
		t.Fatal("partial Boolean proof")
	}
}
