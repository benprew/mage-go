package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentPlayerTargetDomains(t *testing.T) {
	for _, domain := range []struct {
		name           string
		options        []PermanentTargetOption
		control, owner int
	}{
		{"you control", []PermanentTargetOption{TargetYouControl()}, 1, 0},
		{"opponent controls", []PermanentTargetOption{TargetOpponentControls()}, -1, 0},
		{"you own", []PermanentTargetOption{TargetYouOwn()}, 0, 1},
		{"opponent owns", []PermanentTargetOption{TargetOpponentOwns()}, 0, -1},
		{"own and control", []PermanentTargetOption{TargetYouOwn(), TargetYouControl()}, 1, 1},
		{"stolen from opponent", []PermanentTargetOption{TargetOpponentOwns(), TargetYouControl()}, 1, -1},
		{"stolen by opponent", []PermanentTargetOption{TargetYouOwn(), TargetOpponentControls()}, -1, 1},
		{"opponent owns and controls", []PermanentTargetOption{TargetOpponentOwns(), TargetOpponentControls()}, -1, -1},
	} {
		t.Run(domain.name, func(t *testing.T) {
			for _, own := range []bool{false, true} {
				for _, control := range []bool{false, true} {
					g, p, obj, _ := targetQueryFixture(t)
					other := p.ControllerID()
					binding := DeclarePermanentTarget("subject", HasCardTypeFilter(core.TypeLand), domain.options...)
					obj.TargetSpecs[0] = binding.targetSpec()
					p.Card.SetOwner(other)
					if own {
						p.Card.SetOwner(obj.Controller)
					}
					if control {
						p.computedController = obj.Controller
					}
					plan, ok := InspectPermanentTarget(binding)
					if !ok {
						t.Fatal("complete player domain rejected")
					}
					want := (domain.control == 0 || (domain.control == 1) == control) && (domain.owner == 0 || (domain.owner == 1) == own)
					for range 2 {
						got, known := plan.Select(g, obj)
						if !known || (len(got) == 1) != want {
							t.Fatalf("own=%v control=%v: %v/%v want %v", own, control, got, known, want)
						}
						possible := binding.Possible(obj.Controller, obj.Card, g)
						if (len(possible) == 1) != want {
							t.Fatal("native target domain differs")
						}
					}
					binding.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("query called filter") }
					if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
						t.Fatal("callback-free selection failed")
					}
				}
			}
		})
	}
}

func TestPermanentPlayerTargetIdentity(t *testing.T) {
	for _, state := range []string{"control change", "owner change", "phase", "phase cycle", "return", "copy", "copy controller", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, _ := targetQueryFixture(t)
			other := p.ControllerID()
			p.Card.SetOwner(other)
			binding := DeclarePermanentTarget("subject", HasCardTypeFilter(core.TypeLand), TargetOpponentControls(), TargetOpponentOwns())
			obj.TargetSpecs[0] = binding.targetSpec()
			obj.Effects = []Effect{Destroy(binding)}
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("player domain rejected")
			}
			want := true
			switch state {
			case "control change":
				g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: obj.Controller, Duration: core.Indefinite})
				want = false
			case "owner change":
				p.Card.SetOwner(obj.Controller)
				want = false
			case "phase", "phase cycle":
				g.PhaseOut(p.ID())
				want = state == "phase cycle"
				if want {
					g.PhaseIn([]uuid.UUID{p.ID()})
				}
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, other)
				want = false
			case "copy", "copy controller":
				controller := obj.Controller
				if state == "copy controller" {
					controller = other
					want = false
				}
				obj = g.CopyStackObjectDirect(obj, controller, false)
			case "clone":
				clone := g.Clone()
				clone.anteEnabled = true
				if err := clone.ChangeOwner(p.ID(), obj.Controller); err != nil {
					t.Fatal(err)
				}
				if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
					t.Fatal("clone ignored owner change")
				}
			}
			g.ApplyContinuousEffects()
			if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
				t.Fatalf("selection %v/%v want %v", got, known, want)
			}
			if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); !known || certain != want {
				t.Fatalf("destruction %v/%v want %v", certain, known, want)
			}
			if state != "copy controller" {
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
					t.Fatal("native resolution differs")
				}
			}
		})
	}
}

func TestPermanentPlayerTargetUnknown(t *testing.T) {
	for _, option := range []PermanentTargetOption{TargetControlledBy(ActivePlayer), TargetChosenByOpponent()} {
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("restricted", AnyPermanent, TargetYouOwn(), option)); ok {
			t.Fatal("partial player declaration accepted")
		}
	}
}
