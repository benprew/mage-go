package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellDestruction(t *testing.T) {
	for _, state := range []string{"normal", "shield", "no regeneration", "indestructible", "shroud", "hexproof", "protection", "phase", "phase cycle", "return", "type change", "wrong watch", "copy", "hand", "exile", "cancel"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			obj.Effects = []Effect{Destroy(binding)}
			watched := []PermanentVisit{{p.ID(), p.incarnationID}}
			want := true
			switch state {
			case "shield":
				g.AddRegenerationShield(p.ID())
				want = false
			case "no regeneration":
				g.AddRegenerationShield(p.ID())
				obj.Effects = []Effect{DestroyWithoutRegeneration(binding)}
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				want = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				want = false
			case "hexproof":
				p.GrantBaseAttr(core.Hexproof)
				want = false
			case "protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(core.Red))
				want = false
			case "phase":
				g.PhaseOut(p.ID())
				want = false
			case "phase cycle":
				g.PhaseOut(p.ID())
				g.PhaseIn([]uuid.UUID{p.ID()})
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				want = false
			case "type change":
				p.RevokeBaseAttr(core.AttrIsLand)
				p.GrantBaseAttr(core.AttrIsCreature)
				want = false
			case "wrong watch":
				watched[0].Incarnation = uuid.New()
				want = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "hand":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ReturnToHandInsteadOfDyingWithPlayLock())
				want = false
			case "exile":
				g.effects.AddReplacement(&temporaryDeathExileReplacement{subject: p.ID(), incarnation: p.incarnationID})
				want = false
			case "cancel":
				g.effects.AddReplacement(&nextDestructionReplacement{subject: p.ID(), incarnation: p.incarnationID})
				want = false
			}
			before := g.Clone()
			for range 2 {
				certain, known := g.InspectSpellDestruction(obj, watched)
				if !known || certain != want {
					t.Fatalf("certainty=%v/%v want=%v", certain, known, want)
				}
			}
			if !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.effects.replacements, before.effects.replacements) || g.StackSize() != before.StackSize() || p.Tapped {
				t.Fatal("inspection changed game state")
			}
			clone := g.Clone()
			if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), watched); !known || certain != want {
				t.Fatal("clone result differs")
			}
		})
	}
}

func TestSpellDestructionResolutionGate(t *testing.T) {
	for _, state := range []string{"all illegal", "partly legal", "legal prefix", "shield prefix", "illegal prefix"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, first := targetQueryFixture(t)
			second := DeclarePermanentTarget("second", IsLand)
			other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
			obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
			obj.Targets = append(obj.Targets, other.ID())
			obj.targetIncarnations[other.ID()] = other.incarnationID
			obj.TargetZones[other.ID()] = core.ZoneBattlefield
			watched := []PermanentVisit{{other.ID(), other.incarnationID}}
			obj.Effects = []Effect{Sequence(Destroy(first), Destroy(second))}
			want, known := true, true
			switch state {
			case "all illegal":
				p.GrantBaseAttr(core.Shroud)
				other.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{DestroyEach(MatchingPermanents(IsLand))}
				want = false
			case "partly legal":
				p.GrantBaseAttr(core.Shroud)
			case "illegal prefix":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{Sequence(Destroy(first), DestroyEach(MatchingPermanents(IsLand)))}
			case "legal prefix":
				want, known = false, false
			case "shield prefix":
				g.AddRegenerationShield(p.ID())
				want, known = false, false
			}
			if got, gotKnown := g.InspectSpellDestruction(obj, watched); got != want || gotKnown != known {
				t.Fatalf("certainty=%v/%v want=%v/%v", got, gotKnown, want, known)
			}
			g.ResolveStack()
			if state == "all illegal" && g.FindPermanent(other.ID()) == nil {
				t.Fatal("all-illegal spell executed its collection")
			}
			if state != "all illegal" && g.FindPermanent(other.ID()) != nil {
				t.Fatal("legal remaining target did not resolve")
			}
		})
	}
}

func TestSpellDestructionUnknown(t *testing.T) {
	for _, state := range []string{"missing visit", "missing zone", "wrong declaration", "opaque target", "opaque child", "opaque replacement", "cycle replacement", "opaque collection", "unknown effect", "unbound", "ability", "not on stack", "detached", "nil card", "nil target", "duplicate", "missing spec"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			obj.Effects = []Effect{Destroy(binding)}
			trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("inspection ran callback") })
			switch state {
			case "missing visit":
				obj.targetIncarnations = nil
			case "missing zone":
				obj.TargetZones = nil
			case "wrong declaration":
				obj.Effects = []Effect{Destroy(DeclarePermanentTarget("subject", IsLand))}
			case "opaque target":
				binding.spec.(*PermanentTarget).Filters = []PermanentFilter{trap}
			case "opaque child":
				binding.spec.(*PermanentTarget).Filters = []PermanentFilter{Or(AnyPermanent, trap)}
			case "opaque replacement":
				g.effects.AddReplacement(&proposalTrap{})
			case "cycle replacement":
				g.effects.cycleReplacements = []ReplacementEffect{&proposalTrap{}}
			case "opaque collection":
				obj.Effects = []Effect{Sequence(Destroy(binding), DestroyEach(MatchingPermanents(trap)))}
			case "unknown effect":
				obj.Effects = []Effect{Sequence(Destroy(binding), destructionInspectionTrap{})}
			case "unbound":
				obj.Effects = []Effect{Destroy(This)}
			case "ability":
				obj.IsAbility = true
			case "not on stack":
				g.stack.Pop()
			case "detached":
				detached := *obj
				obj = &detached
			case "nil card":
				obj.Card = nil
			case "nil target":
				obj.Targets[0] = uuid.Nil
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, obj.TargetSpecs[0])
				obj.Targets = append(obj.Targets, p.ID())
			case "missing spec":
				obj.TargetSpecs = nil
			}
			if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); certain || known {
				t.Fatal("incomplete declaration supplied an answer")
			}
		})
	}
	var g *Game
	if certain, known := g.InspectSpellDestruction(nil, nil); certain || known {
		t.Fatal("nil query supplied an answer")
	}
}

func TestSpellDestructionNoCallbacks(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	binding.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("typed filter callback") }
	obj.Effects = []Effect{Destroy(binding)}
	if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); !certain || !known {
		t.Fatal("native declaration not inspected")
	}
}
