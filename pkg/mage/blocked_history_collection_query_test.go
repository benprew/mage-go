package mage

import (
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNativeBlockedHistoryCollection(t *testing.T) {
	for _, state := range []string{"normal", "combat reset", "control", "phase attacker", "phase cycle", "return attacker", "depart blocker", "phase blocker", "return blocker", "cleanup", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5), b.PlayerID())
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 5), a.PlayerID())
			g.PutOnBattlefield(NewCreature("Idle", "", 2, 5), a.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
			binding := DeclarePermanentTarget("defender", IsCreature)
			subjects := IntersectPermanents(MatchingPermanents(IsCreature), BlockedByThisTurn(binding))
			query, ok := InspectContextualPermanentCollection(subjects)
			if !ok {
				t.Fatal("native blocking history rejected")
			}
			captured := capturedPermanent{wall.ID(), wall.incarnationID}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{binding.targetDeclaration: {captured}}}
			want := []PermanentVisit{{attacker.ID(), attacker.incarnationID}}
			clone := g.Clone()
			switch state {
			case "combat reset":
				g.GetCombat().Reset()
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase attacker":
				g.PhaseOut(attacker.ID())
				want = nil
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "return attacker":
				g.RemoveFromBattlefield(attacker)
				g.PutOnBattlefield(attacker.Card, a.PlayerID())
				want = nil
			case "depart blocker":
				g.RemoveFromBattlefield(wall)
			case "phase blocker":
				g.PhaseOut(wall.ID())
			case "return blocker":
				g.RemoveFromBattlefield(wall)
				g.PutOnBattlefield(wall.Card, b.PlayerID())
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
				want = nil
			case "clone":
				ctx.Game = clone
				g.RunStepWithPriority(core.Cleanup)
			}
			for range 3 {
				got, known := query.selectKnown(ctx)
				if !known || !reflect.DeepEqual(got, want) {
					t.Fatalf("got %v/%v want %v/true", got, known, want)
				}
				executed := make([]PermanentVisit, 0, len(got))
				for _, p := range subjects.resolvePermanents(ctx) {
					executed = append(executed, PermanentVisit{p.ID(), p.incarnationID})
				}
				if !slices.Equal(got, executed) {
					t.Fatal("native query differs from execution")
				}
				if len(got) > 0 {
					got[0] = PermanentVisit{}
				}
			}
			cloneQuery := *ctx
			cloneQuery.Game = clone
			if got, known := query.selectKnown(&cloneQuery); !known || len(got) != 1 {
				t.Fatal("history mutation escaped into clone")
			}
		})
	}
}

func TestNativeBlockedHistoryCollectionBindings(t *testing.T) {
	g, a, b := newSpellCopyGame()
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5), b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 5), a.PlayerID())
	g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
	binding := DeclarePermanentTarget("defender", IsCreature)
	query, ok := InspectContextualPermanentCollection(BlockedByThisTurn(binding))
	if !ok {
		t.Fatal("native history rejected")
	}
	value := capturedPermanent{wall.ID(), wall.incarnationID}
	for _, tc := range []struct {
		name           string
		values         []capturedPermanent
		present, known bool
		count          int
	}{
		{"legal", []capturedPermanent{value}, true, true, 1},
		{"illegal", nil, true, true, 0},
		{"missing", nil, false, false, 0},
		{"zero id", []capturedPermanent{{incarnation: value.incarnation}}, true, false, 0},
		{"zero visit", []capturedPermanent{{id: value.id}}, true, false, 0},
		{"plural", []capturedPermanent{value, value}, true, false, 0},
		{"foreign visit", []capturedPermanent{{value.id, uuid.New()}}, true, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{}}
			if tc.present {
				ctx.targetBindings[binding.targetDeclaration] = tc.values
			} else {
				ctx.targetBindings[DeclarePermanentTarget("defender", IsCreature).targetDeclaration] = []capturedPermanent{value}
			}
			got, known := query.selectKnown(ctx)
			if known != tc.known || len(got) != tc.count {
				t.Fatalf("got %v/%v", got, known)
			}
			whole, _ := InspectContextualPermanentCollection(UnionPermanents(MatchingPermanents(AnyPermanent), BlockedByThisTurn(binding)))
			if _, known = whole.selectKnown(ctx); known != tc.known {
				t.Fatal("union hid invalid operand")
			}
		})
	}
	var typedNil *PermanentTargetBinding
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("filter callback") })
	for _, ref := range []PermanentRef{nil, typedNil, This, From(BindPermanent("saved")), DeclarePermanentTarget("opaque", opaque), DeclarePermanentTarget("partial", Or(IsCreature, opaque))} {
		for _, set := range []PermanentSet{BlockedByThisTurn(ref), UnionPermanents(MatchingPermanents(AnyPermanent), BlockedByThisTurn(ref))} {
			if _, known := InspectContextualPermanentCollection(set); known {
				t.Fatal("unsupported history operand accepted")
			}
		}
	}
	if _, known := InspectPermanentCollection(BlockedByThisTurn(binding)); known {
		t.Fatal("context-free boundary changed")
	}
}

func TestNativeBlockedHistoryCollectionSpell(t *testing.T) {
	for _, state := range []string{"normal", "regeneration", "no regeneration", "indestructible", "exile", "shroud", "phase wall", "return wall", "missing snapshot", "foreign declaration", "foreign and illegal", "opaque suffix", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5), b.PlayerID())
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 5), a.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
			g.GetCombat().Reset()
			target := DeclarePermanentTarget("wall", IsCreature)
			subjects := BlockedByThisTurn(target)
			filter := IsCreature
			filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
			subjects = IntersectPermanents(MatchingPermanents(filter), subjects)
			effect := (DirectDestructionInstruction{Subjects: subjects, CannotRegenerate: state == "no regeneration"}).Effect()
			card := NewInstant("History", "", WithAbility(Spell(effect, WithTargets(target))))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			target.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("target filter callback") }
			obj := g.StackPeek()
			watched := []PermanentVisit{{attacker.ID(), attacker.incarnationID}}
			certain, known := true, true
			switch state {
			case "regeneration":
				g.AddRegenerationShield(attacker.ID())
				certain = false
			case "no regeneration":
				g.AddRegenerationShield(attacker.ID())
			case "indestructible":
				attacker.GrantBaseAttr(core.Indestructible)
				certain = false
			case "exile":
				if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), attacker.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				certain = false
			case "shroud":
				wall.GrantBaseAttr(core.Shroud)
				certain = false
			case "phase wall":
				g.PhaseOut(wall.ID())
				certain = false
			case "return wall":
				g.RemoveFromBattlefield(wall)
				g.PutOnBattlefield(wall.Card, b.PlayerID())
				certain = false
			case "missing snapshot":
				obj.targetIncarnations = nil
				certain, known = false, false
			case "foreign declaration", "foreign and illegal":
				if state == "foreign and illegal" {
					wall.GrantBaseAttr(core.Shroud)
				}
				obj.Effects = []ResolvingEffect{DestroyEach(BlockedByThisTurn(DeclarePermanentTarget("wall", IsCreature)))}
				certain, known = false, false
			case "opaque suffix":
				obj.Effects = append(obj.Effects, &destructionInspectionTrap{})
				certain, known = false, false
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			random := g.random.Clone()
			stack := g.StackSize()
			before := g.resolution.Clone()
			for range 3 {
				got, k := g.InspectSpellDestruction(obj, watched)
				if got != certain || k != known {
					t.Fatalf("got %v/%v want %v/%v", got, k, certain, known)
				}
			}
			if !reflect.DeepEqual(random, g.random) || g.StackSize() != stack || !reflect.DeepEqual(before, g.resolution.Clone()) {
				t.Fatal("query changed state")
			}
			if state == "regeneration" || state == "no regeneration" {
				if !g.HasRegenerationShield(attacker.ID()) {
					t.Fatal("query consumed shield")
				}
			}
		})
	}
}
