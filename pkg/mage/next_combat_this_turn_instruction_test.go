package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNativeBoundedCombatDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("wall", AnyPermanent)
	body := DestroyEach(FilterPermanentSet(BlockedByThisTurn(target), IsCreature))
	effect := (NextCombatThisTurnInstruction{Body: body}).Effect()
	got, ok := InspectNextCombatThisTurnInstruction(effect)
	if !ok || got.Body != body {
		t.Fatal("lost bounded schedule body")
	}
	if (NextCombatThisTurnInstruction{}).Effect() != nil || (NextCombatThisTurnInstruction{Body: (*sequenceEffect)(nil)}).Effect() != nil {
		t.Fatal("invalid body accepted")
	}
	for _, bad := range []Effect{
		nil, (*nextStepEffect)(nil), AtEach(EachEndOfCombat, body), AtNext(EachEndOfCombat, body),
		AtNextThisTurn(EachEndStep, body), AtNextThisTurn(EachEndOfCombat, nil),
		&nextStepEffect{pattern: EachEndOfCombat, body: body, expiresAtCleanup: true, condition: ControllerLifeAtLeastStartingLife()},
		&nextStepEffect{pattern: EachEndOfCombat, body: body, expiresAtCleanup: true, targets: []Target{target.targetSpec()}},
		&nextStepEffect{pattern: EachEndOfCombat, body: body, expiresAtCleanup: true, modes: []Mode{{}}},
	} {
		if _, ok := InspectNextCombatThisTurnInstruction(bad); ok {
			t.Fatal("invalid schedule accepted")
		}
	}
}

func TestNativeBoundedCombatInspection(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "nested", "singular", "no regeneration", "illegal", "phased", "phase cycle", "returned", "copy", "clone", "foreign history", "foreign singular", "opaque collection", "opaque filter", "opaque child", "unknown suffix", "unknown illegal", "empty", "wrong schedule", "duplicate", "missing snapshot", "missing zone", "foreign suffix", "foreign illegal", "regeneration", "indestructible"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			subjects := FilterPermanentSet(BlockedByThisTurn(target), IsCreature)
			body := DestroyEach(subjects)
			want, known := false, true
			switch state {
			case "nested":
				body = Sequence(Sequence(body), DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature)))
			case "singular":
				body = Destroy(target)
			case "no regeneration":
				body = DestroyEachWithoutRegeneration(subjects)
			case "foreign history", "foreign suffix", "foreign illegal":
				body = DestroyEach(BlockedByThisTurn(DeclarePermanentTarget("foreign", AnyPermanent)))
				known = false
			case "foreign singular":
				body = Destroy(DeclarePermanentTarget("foreign", AnyPermanent))
				known = false
			case "opaque collection":
				body = DestroyEach(collectionQueryTrap{})
				known = false
			case "opaque filter":
				body = DestroyEach(FilterPermanentSet(subjects, NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") })))
				known = false
			case "opaque child", "unknown illegal":
				body = Sequence(body, destructionInspectionTrap{})
				known = false
			case "empty":
				body = Sequence()
				known = false
			}
			delay := (NextCombatThisTurnInstruction{Body: body}).Effect()
			obj.Effects = []Effect{delay}
			switch state {
			case "prefix":
				obj.Effects = []Effect{delay, Destroy(target)}
				want = true
			case "suffix", "copy", "clone":
				obj.Effects = []Effect{Destroy(target), delay}
				want = true
			case "foreign suffix":
				obj.Effects = []Effect{Destroy(target), delay}
			case "foreign illegal":
				p.GrantBaseAttr(core.Shroud)
			case "regeneration":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			case "missing zone":
				obj.TargetZones = nil
				known = false
			case "unknown suffix":
				obj.Effects = []Effect{Destroy(target), delay, destructionInspectionTrap{}}
				known = false
			case "illegal", "unknown illegal":
				p.GrantBaseAttr(core.Shroud)
			case "phased":
				g.PhaseOut(p.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, obj.Controller)
			case "wrong schedule":
				obj.Effects = []Effect{AtNext(EachEndOfCombat, body)}
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				known = false
			case "missing snapshot":
				obj.targetIncarnations = nil
				known = false
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			before := g.resolution.Clone()
			stack := g.StackSize()
			shields := g.HasRegenerationShield(p.ID())
			random := g.random.Clone()
			for range 2 {
				certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if certain != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
				}
			}
			if !reflect.DeepEqual(before, g.resolution.Clone()) || !reflect.DeepEqual(random, g.random) || stack != g.StackSize() || len(g.DelayedTriggers()) != 0 || g.HasRegenerationShield(p.ID()) != shields {
				t.Fatal("inspection changed game state")
			}
		})
	}
}
