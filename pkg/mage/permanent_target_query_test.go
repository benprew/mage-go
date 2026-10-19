package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func targetQueryFixture(t *testing.T) (*Game, *Permanent, *StackObject, *PermanentTargetBinding) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewLand("Subject"), b.PlayerID())
	binding := DeclarePermanentTarget("subject", HasCardTypeFilter(TypeLand))
	card := NewSorcery("Source", "{R}")
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), TargetSpecs: []Target{binding.targetSpec()}, Targets: []uuid.UUID{p.ID()}}
	g.PushStack(obj)
	return g, p, obj, binding
}

func TestPermanentTargetQuery(t *testing.T) {
	for _, state := range []string{"legal", "shroud", "hexproof", "own hexproof", "protection", "different protection", "phase", "return", "type change", "face down", "indestructible", "regeneration"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("typed target rejected")
			}
			want := true
			switch state {
			case "shroud":
				p.GrantBaseAttr(Shroud)
				want = false
			case "hexproof":
				p.GrantBaseAttr(Hexproof)
				want = false
			case "own hexproof":
				p.GrantBaseAttr(Hexproof)
				p.computedController = obj.Controller
			case "protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(Red))
				want = false
			case "different protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(Green))
			case "phase":
				g.PhaseOut(p.ID())
				want = false
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				want = false
			case "type change":
				p.Card = NewCreature("Changed", "", 2, 2)
				want = false
			case "face down":
				p.FaceDown = true
				want = false
			case "indestructible":
				p.GrantBaseAttr(Indestructible)
			case "regeneration":
				g.AddRegenerationShield(p.ID())
			}
			before := g.StackSize()
			got, known := plan.Select(g, obj)
			if !known || (len(got) == 1) != want {
				t.Fatalf("got %v known %v, want member %v", got, known, want)
			}
			if want && got[0] != (PermanentVisit{p.ID(), p.incarnationID}) {
				t.Fatal("wrong visit")
			}
			if len(got) > 0 {
				got[0] = PermanentVisit{}
				again, _ := plan.Select(g, obj)
				if again[0].ID != p.ID() {
					t.Fatal("result aliases query")
				}
			}
			if g.StackSize() != before {
				t.Fatal("query changed stack")
			}
			clone := g.Clone()
			cloned, known := plan.Select(clone, clone.StackPeek())
			original, _ := plan.Select(g, obj)
			if !known || !reflect.DeepEqual(cloned, original) {
				t.Fatal("clone changed query")
			}
		})
	}
}

func TestPermanentTargetQueryUnknown(t *testing.T) {
	trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("query called filter") })
	for _, ref := range []PermanentRef{nil, This, DeclarePermanentTarget("opaque", trap), DeclarePermanentTarget("partial", Or(AnyPermanent, trap)), DeclarePermanentTarget("control", HasCardTypeFilter(TypeLand), TargetControlledBy(ActivePlayer))} {
		if _, ok := InspectPermanentTarget(ref); ok {
			t.Fatal("unknown declaration accepted")
		}
	}
	var nilTarget *PermanentTargetBinding
	if _, ok := InspectPermanentTarget(nilTarget); ok {
		t.Fatal("nil binding accepted")
	}
	g, p, obj, binding := targetQueryFixture(t)
	plan, _ := InspectPermanentTarget(binding)
	other := DeclarePermanentTarget("subject", HasCardTypeFilter(TypeLand))
	otherPlan, _ := InspectPermanentTarget(other)
	if _, known := otherPlan.Select(g, obj); known {
		t.Fatal("same name substituted for declaration identity")
	}
	var zero PermanentTargetPredicate
	for _, tc := range []struct {
		g   *Game
		obj *StackObject
	}{{nil, obj}, {g, nil}} {
		if _, known := plan.Select(tc.g, tc.obj); known {
			t.Fatal("missing context accepted")
		}
	}
	if _, known := zero.Select(g, obj); known {
		t.Fatal("zero plan accepted")
	}
	saved := obj.targetIncarnations
	obj.targetIncarnations = nil
	if _, known := plan.Select(g, obj); known {
		t.Fatal("missing announced visit guessed")
	}
	obj.targetIncarnations = saved
	p.RuntimeAbilities = append(p.RuntimeAbilities, &ProtectionAbility{Filter: NewCardFilter("opaque", func(Card) bool { panic("query called protection") })})
	if _, known := plan.Select(g, obj); known {
		t.Fatal("opaque protection accepted")
	}
}

func TestPermanentTargetQueryNoCallbacks(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	spec := binding.spec.(*PermanentTarget)
	spec.Filters[0].fn = func(*Permanent, *Game) bool { panic("query called typed filter") }
	plan, ok := InspectPermanentTarget(binding)
	if !ok {
		t.Fatal("typed predicate lost")
	}
	p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(Green))
	g.random.SetRandomResults([]int{3, 1, 4})
	beforeRandom := g.random.Clone()
	before := g.Clone()
	got, known := plan.Select(g, obj)
	if !known || len(got) != 1 || g.StackSize() != before.StackSize() || !reflect.DeepEqual(g.random, beforeRandom) {
		t.Fatal("pure query failed")
	}
	g.effects.effects = append(g.effects.effects, &spellTargetRestriction{filter: NewCardFilter("trap", func(Card) bool { panic("query ran restriction") })})
	if _, known := plan.Select(g, obj); known {
		t.Fatal("uninspected global restriction accepted")
	}
}

func TestPermanentTargetQuerySourceInformation(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePermanentTarget("subject", HasCardTypeFilter(TypeLand))
			card := NewCreature("Source", "{G}", 2, 2, WithAbility(Activate(Costs(ManaPayment("{0}")), Destroy(target), WithTargets(target))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewLand("Subject"), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			g.ChangeObjectColor(source.ID(), Blue)
			switch state {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "returned" {
					g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "copy":
				obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
			}
			current := g.FindPermanent(victim.ID())
			current.RuntimeAbilities = append(current.RuntimeAbilities, ProtectionFromColor(Blue))
			plan, ok := InspectPermanentTarget(target)
			if !ok {
				t.Fatal("ability declaration rejected")
			}
			if got, known := plan.Select(g, obj); !known || len(got) != 0 {
				t.Fatalf("lost source color: %v %v", got, known)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) == nil {
				t.Fatal("native legality differs")
			}
		})
	}
}

func TestPermanentTargetQueryIndependentGroups(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	other := DeclarePermanentTarget("other", HasCardTypeFilter(TypeLand))
	second := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
	obj.TargetSpecs = append(obj.TargetSpecs, other.targetSpec())
	obj.Targets = append(obj.Targets, second.ID())
	obj.targetIncarnations[second.ID()] = second.incarnationID
	obj.TargetZones[second.ID()] = ZoneBattlefield
	p.GrantBaseAttr(Shroud)
	firstPlan, _ := InspectPermanentTarget(binding)
	secondPlan, _ := InspectPermanentTarget(other)
	if got, known := firstPlan.Select(g, obj); !known || len(got) != 0 {
		t.Fatal("illegal group selected")
	}
	if got, known := secondPlan.Select(g, obj); !known || len(got) != 1 || got[0].ID != second.ID() {
		t.Fatal("independent group lost")
	}
	copied := g.CopyStackObjectDirect(obj, obj.Controller, false)
	if got, known := secondPlan.Select(g, copied); !known || len(got) != 1 {
		t.Fatal("copy lost selected group")
	}
	delete(copied.targetIncarnations, second.ID())
	if _, known := secondPlan.Select(g, copied); known {
		t.Fatal("copy guessed missing identity")
	}
	if got, known := secondPlan.Select(g, obj); !known || len(got) != 1 {
		t.Fatal("copy changed original identity")
	}
}

func TestPermanentTargetQueryPhaseCycleAndLayers(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	plan, _ := InspectPermanentTarget(binding)
	original, known := plan.Select(g, obj)
	if !known || len(original) != 1 {
		t.Fatal("initial target missing")
	}
	g.PhaseOut(p.ID())
	if got, known := plan.Select(g, obj); !known || len(got) != 0 {
		t.Fatal("phased target present")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if got, known := plan.Select(g, obj); !known || !reflect.DeepEqual(got, original) {
		t.Fatal("phase cycle changed visit")
	}
	layered := NewArtifact("Layered", "", WithAbility(Static(While(PermanentMatchesFilter(This, IsArtifact), AddCardType(This, TypeLand)))))
	other := g.PutOnBattlefield(layered, obj.Controller)
	obj.Targets = []uuid.UUID{other.ID()}
	obj.targetIncarnations = map[uuid.UUID]uuid.UUID{other.ID(): other.incarnationID}
	obj.TargetZones = map[uuid.UUID]Zone{other.ID(): ZoneBattlefield}
	if got, known := plan.Select(g, obj); !known || len(got) != 1 || got[0].ID != other.ID() {
		t.Fatal("current type layer ignored")
	}
}
