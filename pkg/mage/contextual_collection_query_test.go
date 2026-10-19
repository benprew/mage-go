package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestContextualCollectionQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ownCard := NewLand("Owned")
	ownCard.SetOwner(a.PlayerID())
	own := g.PutOnBattlefield(ownCard, b.PlayerID())
	foreignCard := NewLand("Foreign")
	foreignCard.SetOwner(b.PlayerID())
	foreign := g.PutOnBattlefield(foreignCard, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Creature", "{B}", 2, 2), a.PlayerID())
	sets := []PermanentSet{
		ControlledPermanents(IsLand), PermanentsControlledBy(You, IsLand), PermanentsControlledBy(ActivePlayer, IsLand),
		PermanentsOwnedBy(You, IsLand), PermanentsOwnedBy(ActivePlayer, IsLand),
		IntersectPermanents(MatchingPermanents(IsLand), PermanentsControlledBy(You, AnyPermanent)),
		UnionPermanents(PermanentsOwnedBy(You, IsLand), PermanentsControlledBy(You, IsLand), PermanentsOwnedBy(You, IsLand)),
		IntersectPermanents(PermanentsControlledBy(You, AnyPermanent), MatchingPermanents(Not(HasColorFilter(core.Black)))),
	}
	for _, set := range sets {
		plan, ok := InspectContextualPermanentCollection(set)
		if !ok {
			t.Fatal("missing contextual plan")
		}
		for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			ctx := &EffectContext{Game: g, Controller: controller}
			var want []PermanentVisit
			for _, p := range set.resolvePermanents(ctx) {
				want = append(want, PermanentVisit{p.ID(), p.incarnationID})
			}
			if got := plan.Select(ctx); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		}
	}
	plan, ok := InspectContextualPermanentCollection(PermanentsControlledBy(You, IsLand))
	if !ok {
		t.Fatal("missing plan")
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	before := plan.Select(ctx)
	clone := g.Clone()
	clone.AddControlEffect(ControlEffectSpec{SourceID: own.ID(), TargetID: foreign.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if len(plan.Select(&EffectContext{Game: clone, Controller: a.PlayerID()})) != 0 || !reflect.DeepEqual(before, plan.Select(ctx)) {
		t.Fatal("control change or clone isolation")
	}
	g.PhaseOut(foreign.ID())
	if len(plan.Select(ctx)) != 0 {
		t.Fatal("phased member")
	}
	g.PhaseIn([]uuid.UUID{foreign.ID()})
	if !reflect.DeepEqual(before, plan.Select(ctx)) {
		t.Fatal("phase changed visit")
	}
	g.RemoveFromBattlefield(foreign)
	returned := g.PutOnBattlefield(foreign.Card, a.PlayerID())
	after := plan.Select(ctx)
	if len(after) != 1 || after[0].ID != returned.ID() || after[0].Incarnation == before[0].Incarnation {
		t.Fatal("return reused visit")
	}
	g.RemoveFromBattlefield(returned)
	if len(plan.Select(ctx)) != 0 {
		t.Fatal("query used departure information")
	}
}

type contextualPlayerTrap struct{ PlayerRef }

func (contextualPlayerTrap) resolvePlayer(*EffectContext) (uuid.UUID, bool) { panic("player callback") }

func TestContextualCollectionQueryNoCallbacks(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	filter := IsLand
	filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
	members := []PermanentSet{PermanentsControlledBy(You, filter), PermanentsControlledBy(ActivePlayer, filter)}
	set := UnionPermanents(members...)
	members[0] = collectionQueryTrap{}
	plan, ok := InspectContextualPermanentCollection(set)
	if !ok {
		t.Fatal("missing plan")
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	random := g.random.Clone()
	for range 3 {
		got := plan.Select(ctx)
		if len(got) != 1 || got[0].ID != p.ID() {
			t.Fatal("duplicates or lost operands")
		}
		got[0] = PermanentVisit{}
	}
	if !reflect.DeepEqual(random, g.random) || g.StackSize() != 0 {
		t.Fatal("query changed game")
	}
	g.SetActivePlayerIndex(1)
	active, _ := InspectContextualPermanentCollection(PermanentsControlledBy(ActivePlayer, filter))
	if len(active.Select(ctx)) != 0 {
		t.Fatal("active player was captured too soon")
	}
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque filter") })
	for _, unknown := range []PermanentSet{
		nil, collectionQueryTrap{}, PermanentsControlledBy(nil, IsLand), PermanentsOwnedBy(contextualPlayerTrap{}, IsLand),
		PermanentsControlledBy(You, opaque), PermanentsOwnedBy(You, opaque),
		PermanentsControlledBy(FromPlayer(BindPlayer("unsupported")), IsLand),
		UnionPermanents(MatchingPermanents(AnyPermanent), PermanentsControlledBy(contextualPlayerTrap{}, IsLand)),
		IntersectPermanents(MatchingPermanents(Or()), collectionQueryTrap{}), UnionPermanents(),
	} {
		if _, ok := InspectContextualPermanentCollection(unknown); ok {
			t.Fatal("partial query")
		}
	}
	if plan.Select(nil) != nil || plan.Select(&EffectContext{}) != nil || plan.Select(&EffectContext{Game: g}) != nil {
		t.Fatal("missing context selected members")
	}
	var zero ContextualPermanentCollectionPredicate
	if zero.Select(ctx) != nil {
		t.Fatal("zero plan selected members")
	}
	if _, ok := InspectPermanentCollection(set); ok {
		t.Fatal("context-free boundary changed")
	}
}
