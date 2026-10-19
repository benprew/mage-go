package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSubtypeCollection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dual := g.PutOnBattlefield(NewLand("Dual", WithSubTypes("Forest", "Island")), a.PlayerID())
	g.PutOnBattlefield(NewLand("Desert", WithSubTypes("Desert")), b.PlayerID())
	g.PutOnBattlefield(NewCreature("Elf", "", 2, 2, WithSubTypes("Elf")), b.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), a.PlayerID())
	phased := g.PutOnBattlefield(NewLand("Phased", WithSubTypes("Forest")), b.PlayerID())
	g.PhaseOut(phased.ID())
	for _, filter := range []PermanentFilter{
		HasSubType("Forest"), Not(HasSubType("Forest")),
		And(IsLand, HasSubType("Island")), Or(IsCreature, HasSubType("Desert")),
		Or(HasSubType("Forest"), HasSubType("Island")), HasSubType("Elf"), HasSubType("Aura"),
		And(Or(HasSubType("Island"), IsCreature), Not(HasSubType("Elf"))),
	} {
		plan, ok := InspectPermanentCollection(MatchingPermanents(filter))
		if !ok {
			t.Fatalf("missing subtype plan: %s", filter.Text())
		}
		var want []PermanentVisit
		for _, p := range MatchingPermanents(filter).resolvePermanents(&EffectContext{Game: g}) {
			want = append(want, PermanentVisit{ID: p.ID(), Incarnation: p.incarnationID})
		}
		if got := plan.Select(g); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %v, want %v", filter.Text(), got, want)
		}
	}
	plan, ok := InspectPermanentCollection(MatchingPermanents(HasSubType("Forest")))
	if !ok {
		t.Fatal("missing Forest plan")
	}
	before := plan.Select(g)
	clone := g.Clone()
	clone.AddContinuousEffect(SetSubtypes(dual.ID(), SubtypeLand, core.EndOfTurn, "Mountain"))
	if len(plan.Select(clone)) != 0 || !reflect.DeepEqual(before, plan.Select(g)) {
		t.Fatal("current subtype or clone isolation failed")
	}
	g.PhaseOut(dual.ID())
	if len(plan.Select(g)) != 0 {
		t.Fatal("phased land remained selected")
	}
	g.PhaseIn([]uuid.UUID{dual.ID()})
	if !reflect.DeepEqual(before, plan.Select(g)) {
		t.Fatal("phase cycle changed the visit")
	}
	g.RemoveFromBattlefield(dual)
	returned := g.PutOnBattlefield(dual.Card, b.PlayerID())
	after := plan.Select(g)
	if len(after) != 1 || after[0].ID != returned.ID() || after[0].Incarnation == before[0].Incarnation {
		t.Fatal("returned land kept its old visit")
	}
	returned.FaceDown = true
	if len(plan.Select(g)) != 0 {
		t.Fatal("face-down subtype remained selected")
	}
}

func TestPermanentSubtypeCollectionNoCallbacks(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewLand("Dual", WithSubTypes("Forest", "Island")), a.PlayerID())
	filter := HasSubType("Forest")
	filter.fn = func(*Permanent, *Game) bool { panic("typed filter callback") }
	parts := []PermanentFilter{filter, IsLand}
	composed := And(parts...)
	parts[0] = IsCreature
	plan, ok := InspectPermanentCollection(UnionPermanents(MatchingPermanents(composed), MatchingPermanents(filter)))
	if !ok {
		t.Fatal("missing composed subtype plan")
	}
	g.AddRegenerationShield(p.ID())
	p.GrantBaseAttr(core.Indestructible)
	beforeRandom := g.random.Clone()
	for range 3 {
		visits := plan.Select(g)
		if len(visits) != 1 || visits[0].ID != p.ID() {
			t.Fatal("selection inferred destruction or lost immutable operands")
		}
		visits[0] = PermanentVisit{}
	}
	if !reflect.DeepEqual(beforeRandom, g.random) || g.StackSize() != 0 {
		t.Fatal("inspection changed game state")
	}
	trap := NewPermanentFilter("Forest", func(*Permanent, *Game) bool { panic("opaque callback") })
	for _, filter := range []PermanentFilter{trap, Not(trap), And(filter, trap), Or(AnyPermanent, trap), Or(filter, trap)} {
		if _, ok := InspectPermanentCollection(MatchingPermanents(filter)); ok {
			t.Fatal("opaque branch supplied a partial query")
		}
	}
	if _, ok := InspectPermanentCardTypeFilter(HasSubType("Forest")); ok {
		t.Fatal("subtype predicate became a type-only predicate")
	}
}
