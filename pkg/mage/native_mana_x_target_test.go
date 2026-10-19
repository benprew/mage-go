package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNativeManaXTargetPermanent(t *testing.T) {
	for _, state := range []string{"match", "mismatch", "zero", "multiple X", "negative X", "cost change", "shroud", "phase", "phase cycle", "return", "missing visit", "missing zone", "foreign", "duplicate", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			target.spec.(*PermanentTarget).manaValueEqualX = true
			p.Card.(*BaseCard).manaCost = core.ParseManaCost("{2}")
			obj.XValue = 2
			obj.Effects = []Effect{Destroy(target)}
			want, known := true, true
			switch state {
			case "mismatch":
				obj.XValue = 3
				want = false
			case "zero":
				p.Card.(*BaseCard).manaCost = core.ParseManaCost("{X}")
				obj.XValue = 0
			case "multiple X":
				p.Card.(*BaseCard).manaCost = core.ParseManaCost("{X}{X}{2}")
			case "negative X":
				obj.XValue = -1
				want, known = false, false
			case "cost change":
				p.Card.(*BaseCard).manaCost = core.ParseManaCost("{3}")
				want = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				want = false
			case "phase", "phase cycle":
				g.PhaseOut(p.ID())
				if state == "phase cycle" {
					g.PhaseIn([]uuid.UUID{p.ID()})
				} else {
					want = false
				}
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				want = false
			case "missing visit":
				delete(obj.targetIncarnations, p.ID())
				want, known = false, false
			case "missing zone":
				delete(obj.TargetZones, p.ID())
				want, known = false, false
			case "foreign":
				target = DeclarePermanentTarget("foreign", IsLand, TargetManaValueEqualX())
				want, known = false, false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				want, known = false, false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			}
			predicate, ok := InspectPermanentTarget(target)
			if !ok {
				t.Fatal("native announced-X restriction rejected")
			}
			g.resolution.SetX(77)
			for range 2 {
				visits, k := predicate.Select(g, obj)
				if k != known || (len(visits) == 1) != want {
					t.Fatalf("selected=%v known=%v want=%v/%v", visits, k, want, known)
				}
				if g.resolution.X() != 77 || g.StackSize() == 0 {
					t.Fatal("query changed resolution frame or stack")
				}
			}
			if state == "clone" {
				clone := g.Clone()
				clone.StackPeek().XValue = 3
				if visits, k := predicate.Select(clone, clone.StackPeek()); !k || len(visits) != 0 {
					t.Fatal("clone read original X")
				}
			}
			if state == "match" || state == "multiple X" || state == "zero" || state == "copy" {
				if certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); !k || !certain {
					t.Fatal("native target did not permit direct destruction proof")
				}
				g.AddRegenerationShield(p.ID())
				for range 2 {
					if certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); !k || certain {
						t.Fatal("shield ignored")
					}
				}
				if p.Tapped || g.resolution.X() != 77 {
					t.Fatal("inspection used shield or frame")
				}
			}
		})
	}
}

func TestNativeManaXTargetSpell(t *testing.T) {
	for _, state := range []string{"match", "mismatch", "negative X", "multiple X", "zero", "cost change", "target X change", "left", "recast", "missing zone", "foreign", "duplicate", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, victim, obj, target := counterInstructionFixture(t)
			target.spec.(*typedSpellTarget).manaValueEqualX = true
			victim.Card.(*BaseCard).manaCost = core.ParseManaCost("{X}{X}{U}")
			victim.XValue = 2
			obj.XValue = 5
			want, known := true, true
			switch state {
			case "mismatch":
				obj.XValue = 2
				want = false
			case "negative X":
				obj.XValue = -1
				want, known = false, false
			case "multiple X":
				victim.XValue = 3
				obj.XValue = 7
			case "zero":
				victim.Card.(*BaseCard).manaCost = core.ParseManaCost("{X}")
				victim.XValue = 0
				obj.XValue = 0
			case "cost change":
				victim.Card.(*BaseCard).manaCost = core.ParseManaCost("{X}{U}")
				want = false
			case "target X change":
				victim.XValue = 3
				want = false
			case "left", "recast":
				g.CounterStackObject(victim.ID)
				want = false
				if state == "recast" {
					replacement := *victim
					replacement.ID = uuid.New()
					g.PushStack(&replacement)
				}
			case "missing zone":
				delete(obj.TargetZones, victim.ID)
				want, known = false, false
			case "foreign":
				target = DeclareSpellTargetManaValueX("foreign")
				want, known = false, false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, victim.ID)
				want, known = false, false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			}
			predicate, ok := InspectSpellTarget(target)
			if !ok {
				t.Fatal("native spell X restriction rejected")
			}
			g.resolution.SetX(77)
			before := g.Clone()
			for range 2 {
				id, k := predicate.Select(g, obj)
				if k != known || (id == victim.ID) != want {
					t.Fatalf("selected=%v known=%v want=%v/%v", id, k, want, known)
				}
			}
			if !reflect.DeepEqual(g.random, before.random) || g.resolution.X() != 77 || g.StackSize() != before.StackSize() {
				t.Fatal("query changed state")
			}
			if state == "clone" {
				clone := g.Clone()
				clone.StackPeek().XValue = 7
				if id, k := predicate.Select(clone, clone.StackPeek()); !k || id != uuid.Nil {
					t.Fatal("clone read original X")
				}
			}
			if state == "match" || state == "zero" || state == "copy" {
				if certain, k := g.InspectSpellDestruction(obj, nil); certain || !k {
					t.Fatal("complete counter tree rejected")
				}
			}
		})
	}
}

func TestNativeManaXTargetInvalid(t *testing.T) {
	for _, target := range []Target{nil, (*targetDeclaration)(nil), TargetPermanent(IsLand)} {
		if _, ok := InspectPermanentTargetDeclaration(target); ok {
			t.Fatal("undeclared target accepted")
		}
	}
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { t.Fatal("inspection ran callback"); return true })
	if _, ok := InspectPermanentTarget(DeclarePermanentTarget("opaque", opaque, TargetManaValueEqualX())); ok {
		t.Fatal("opaque target accepted")
	}
	spell := DeclareSpellTargetManaValueX("opaque", NewCardFilter("opaque", func(Card) bool { t.Fatal("inspection ran callback"); return true }))
	if _, ok := InspectSpellTarget(spell); ok {
		t.Fatal("opaque spell accepted")
	}
	var permanentNil *PermanentTargetBinding
	var spellNil *SpellTargetBinding
	if _, ok := InspectPermanentTarget(permanentNil); ok {
		t.Fatal("nil permanent accepted")
	}
	if _, ok := InspectSpellTarget(spellNil); ok {
		t.Fatal("nil spell accepted")
	}
}

func TestNativeManaXTargetPlural(t *testing.T) {
	for _, fixedCount := range []bool{false, true} {
		g, first, second, obj, binding := pluralQueryFixture(t, 2)
		if fixedCount {
			binding = DeclarePermanentTargets("fixed", 2, 2, IsCreature, TargetManaValueEqualX())
		} else {
			binding.spec.(*countBoundTarget).inner.(*PermanentTarget).manaValueEqualX = true
		}
		obj.TargetSpecs = []Target{binding.targetSpec(), binding.targetSpec()}
		obj.Effects = []Effect{DestroyEach(binding)}
		first.Card.(*BaseCard).manaCost = core.ParseManaCost("{2}")
		second.Card.(*BaseCard).manaCost = core.ParseManaCost("{3}")
		predicate, ok := InspectPermanentTargets(binding)
		if !ok {
			t.Fatal("native plural X restriction rejected")
		}
		g.resolution.SetX(77)
		for range 2 {
			visits, known := predicate.Select(g, obj)
			if !known || len(visits) != 1 || visits[0].ID != first.ID() {
				t.Fatal("plural restriction did not check each current mana value")
			}
			if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{first.ID(), first.incarnationID}}); !known || !certain {
				t.Fatal("partly legal group lost its direct proof")
			}
			if g.resolution.X() != 77 {
				t.Fatal("plural inspection changed X")
			}
		}
		obj.XValue = -1
		if _, known := predicate.Select(g, obj); known {
			t.Fatal("negative X accepted")
		}
	}
}
