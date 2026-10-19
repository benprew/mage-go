package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNativeTargetControllerRecipient(t *testing.T) {
	for _, state := range []string{"normal", "control", "phase", "cycle", "leave", "return", "shroud", "indestructible", "regeneration", "missing visit", "missing zone", "duplicate", "foreign", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			query, ok := InspectDamageRecipient(ControllerOfWithLKI(target))
			if !ok {
				t.Fatal("native target controller rejected")
			}
			want, known := p.ControllerID(), true
			switch state {
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: obj.Controller, Duration: core.Indefinite})
				want = obj.Controller
			case "phase":
				g.PhaseOut(p.ID())
				want = uuid.Nil
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "leave", "return":
				g.RemoveFromBattlefield(p)
				if state == "return" {
					g.PutOnBattlefield(p.Card, obj.Controller)
				}
				want = uuid.Nil
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				want = uuid.Nil
			case "indestructible":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
			case "regeneration":
				g.AddRegenerationShield(p.ID())
			case "missing visit":
				delete(obj.targetIncarnations, p.ID())
				known = false
			case "missing zone":
				delete(obj.TargetZones, p.ID())
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, obj.TargetSpecs[0])
				obj.Targets = append(obj.Targets, obj.Targets[0])
				known = false
			case "foreign":
				query, _ = InspectDamageRecipient(ControllerOfWithLKI(DeclarePermanentTarget("subject", IsLand)))
				known = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, p.ControllerID(), false)
			case "clone":
				clone := g.Clone()
				g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: obj.Controller, Duration: core.Indefinite})
				g, obj = clone, clone.StackPeek()
			}
			g.ApplyContinuousEffects()
			random, frame, stack := g.random.Clone(), g.resolution.Clone(), g.StackSize()
			for range 3 {
				got, k := query.Select(g, obj)
				if k != known || k && (got.Player != want || got.Permanent != (PermanentVisit{})) {
					t.Fatalf("got %+v/%v want %v/%v", got, k, want, known)
				}
			}
			if !reflect.DeepEqual(random, g.random) || !reflect.DeepEqual(frame, g.resolution.Clone()) || g.StackSize() != stack {
				t.Fatal("inspection changed state")
			}
		})
	}
}

func TestNativeTargetControllerRecipientInvalid(t *testing.T) {
	var missing *PermanentTargetBinding
	for _, ref := range []PermanentRef{nil, missing, This, AttachedPermanent, From(BindPermanent("saved")), DeclarePermanentTarget("opaque", NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter callback") }))} {
		if _, ok := InspectDamageRecipient(ControllerOfWithLKI(ref)); ok {
			t.Fatalf("unsupported controller accepted: %T", ref)
		}
	}
	if _, ok := InspectDamageRecipient(OwnerOfWithLKI(DeclarePermanentTarget("owner", IsLand))); ok {
		t.Fatal("owner requires a separate inspection contract")
	}
}

func TestNativeTargetControllerRecipientSpell(t *testing.T) {
	for _, state := range []string{"suffix", "prefix", "zero prefix", "illegal prefix", "all illegal", "foreign", "foreign all illegal", "unknown suffix", "unknown all illegal", "regeneration", "indestructible", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
			second := DeclarePermanentTarget("other", IsLand)
			g.stack.Pop()
			obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
			obj.Targets = append(obj.Targets, other.ID())
			g.PushStack(obj)
			ref := target
			if state == "foreign" || state == "foreign all illegal" {
				ref = DeclarePermanentTarget("subject", IsLand)
			}
			amount := 2
			if state == "zero prefix" {
				amount = 0
			}
			damage := DealDamageFrom(This, ControllerOfWithLKI(ref), Fixed(amount))
			obj.Effects = []Effect{Sequence(Destroy(second), damage)}
			want, known := true, true
			switch state {
			case "prefix", "zero prefix", "illegal prefix", "all illegal":
				obj.Effects = []Effect{Sequence(damage, Destroy(second))}
				known, want = false, false
				if state == "illegal prefix" || state == "all illegal" {
					p.GrantBaseAttr(core.Shroud)
					known, want = true, true
				}
				if state == "all illegal" {
					other.GrantBaseAttr(core.Shroud)
					want = false
				}
			case "foreign", "foreign all illegal":
				known, want = false, false
			case "unknown suffix", "unknown all illegal":
				obj.Effects = append(obj.Effects, destructionInspectionTrap{})
				known, want = false, false
			case "regeneration":
				g.AddRegenerationShield(other.ID())
				want = false
			case "indestructible":
				other.GrantBaseAttr(core.Indestructible)
				want = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, p.ControllerID(), false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			}
			if state == "foreign all illegal" || state == "unknown all illegal" {
				p.GrantBaseAttr(core.Shroud)
				other.GrantBaseAttr(core.Shroud)
			}
			g.ApplyContinuousEffects()
			random, frame, stack := g.random.Clone(), g.resolution.Clone(), g.StackSize()
			for range 3 {
				got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{other.ID(), other.incarnationID}})
				if got != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", got, k, want, known)
				}
			}
			if !reflect.DeepEqual(random, g.random) || !reflect.DeepEqual(frame, g.resolution.Clone()) || g.StackSize() != stack {
				t.Fatal("query changed state")
			}
			if state == "regeneration" && !g.HasRegenerationShield(other.ID()) {
				t.Fatal("query consumed a shield")
			}
		})
	}
}
