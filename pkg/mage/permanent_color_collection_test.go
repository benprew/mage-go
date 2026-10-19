package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentColorCollection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	black := g.PutOnBattlefield(NewCreature("Black", "{B}", 2, 2), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Multicolor", "{B}{W}", 2, 2), b.PlayerID())
	g.PutOnBattlefield(NewCreature("Colorless", "", 2, 2), b.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Black enchantment", "{B}"), a.PlayerID())
	for _, filter := range []PermanentFilter{
		HasColorFilter(core.Black), Not(HasColorFilter(core.Black)),
		And(IsCreature, HasColorFilter(core.Black)),
		Or(HasColorFilter(core.White), HasColorFilter(core.Black)),
		And(IsCreature, Not(HasColorFilter(core.Black))),
		And(Or(IsEnchantment, HasColorFilter(core.Black)), Not(HasSubType("Wall"))),
	} {
		plan, ok := InspectPermanentCollection(MatchingPermanents(filter))
		if !ok {
			t.Fatalf("missing color plan: %s", filter.Text())
		}
		var want []PermanentVisit
		for _, p := range MatchingPermanents(filter).resolvePermanents(&EffectContext{Game: g}) {
			want = append(want, PermanentVisit{p.ID(), p.incarnationID})
		}
		if got := plan.Select(g); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: %v != %v", filter.Text(), got, want)
		}
	}
	plan, ok := InspectPermanentCollection(MatchingPermanents(And(IsCreature, HasColorFilter(core.Black))))
	if !ok {
		t.Fatal("missing plan")
	}
	before := plan.Select(g)
	clone := g.Clone()
	clone.AddContinuousEffect(BecomesColor(black.ID(), core.White, core.EndOfTurn))
	if len(plan.Select(clone)) != 1 || !reflect.DeepEqual(before, plan.Select(g)) {
		t.Fatal("layer or clone isolation")
	}
	g.PhaseOut(black.ID())
	if len(plan.Select(g)) != 1 {
		t.Fatal("phased member")
	}
	g.PhaseIn([]uuid.UUID{black.ID()})
	if !reflect.DeepEqual(before, plan.Select(g)) {
		t.Fatal("phase changed visit")
	}
	g.RemoveFromBattlefield(black)
	returned := g.PutOnBattlefield(black.Card, b.PlayerID())
	after := plan.Select(g)
	if len(after) != 2 || after[1].ID != returned.ID() || after[1].Incarnation == before[0].Incarnation {
		t.Fatal("returned visit")
	}
	returned.FaceDown = true
	if len(plan.Select(g)) != 1 {
		t.Fatal("face-down color")
	}
}

func TestPermanentColorCollectionNoCallbacks(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Black", "{B}", 2, 2), a.PlayerID())
	filter := HasColorFilter(core.Black)
	filter.fn = func(*Permanent, *Game) bool { panic("typed callback") }
	parts := []PermanentFilter{filter, IsCreature}
	combined := And(parts...)
	parts[0] = IsLand
	plan, ok := InspectPermanentCollection(UnionPermanents(MatchingPermanents(combined), MatchingPermanents(filter)))
	if !ok {
		t.Fatal("missing color plan")
	}
	random := g.random.Clone()
	for range 3 {
		visits := plan.Select(g)
		if len(visits) != 1 || visits[0].ID != p.ID() {
			t.Fatal("lost immutable operands or duplicate member")
		}
		visits[0] = PermanentVisit{}
	}
	if !reflect.DeepEqual(random, g.random) || g.StackSize() != 0 {
		t.Fatal("query changed state")
	}
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque callback") })
	for _, f := range []PermanentFilter{opaque, Not(opaque), And(filter, opaque), Or(AnyPermanent, opaque), Or(filter, opaque)} {
		if _, ok := InspectPermanentCollection(MatchingPermanents(f)); ok {
			t.Fatal("partial query")
		}
	}
	if _, ok := InspectPermanentCardTypeFilter(filter); ok {
		t.Fatal("color became type-only")
	}
}
