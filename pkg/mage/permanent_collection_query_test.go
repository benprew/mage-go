package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentCollectionQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	both := g.PutOnBattlefield(NewArtifact("Both", "", WithAbility(Static(While(PermanentMatchesFilter(This, IsArtifact), AddCardType(This, core.TypeCreature))))), b.PlayerID())
	phased := g.PutOnBattlefield(NewLand("Phased"), b.PlayerID())
	g.PhaseOut(phased.ID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, set := range []PermanentSet{
		MatchingPermanents(AnyPermanent), MatchingPermanents(IsLand), MatchingPermanents(And(IsArtifact, IsCreature)),
		MatchingPermanents(Not(IsCreature)), UnionPermanents(MatchingPermanents(IsLand), MatchingPermanents(IsCreature), MatchingPermanents(IsLand)),
		IntersectPermanents(MatchingPermanents(IsArtifact), MatchingPermanents(IsCreature)),
	} {
		plan, ok := InspectPermanentCollection(set)
		if !ok {
			t.Fatal("known collection rejected")
		}
		got := plan.Select(g)
		var want []PermanentVisit
		for _, p := range set.resolvePermanents(ctx) {
			want = append(want, PermanentVisit{ID: p.ID(), Incarnation: p.incarnationID})
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if len(got) > 0 {
			got[0] = PermanentVisit{}
			if reflect.DeepEqual(got, plan.Select(g)) {
				t.Fatal("caller changed query state")
			}
		}
	}
	plan, ok := InspectPermanentCollection(MatchingPermanents(IsLand))
	if !ok {
		t.Fatal("land plan rejected")
	}
	before := plan.Select(g)
	clone := g.Clone()
	clone.PhaseOut(land.ID())
	if len(plan.Select(clone)) != 0 || !reflect.DeepEqual(before, plan.Select(g)) {
		t.Fatal("clone query changed original")
	}
	g.RemoveFromBattlefield(land)
	returned := g.PutOnBattlefield(land.Card, a.PlayerID())
	after := plan.Select(g)
	if len(after) != 1 || after[0].ID != returned.ID() || after[0].Incarnation == before[0].Incarnation {
		t.Fatal("return reused a visit")
	}
	g.FindPermanent(creature.ID()).FaceDown = true
	g.FindPermanent(both.ID()).FaceDown = true
	artifacts, ok := InspectPermanentCollection(MatchingPermanents(IsArtifact))
	if !ok || len(artifacts.Select(g)) != 0 {
		t.Fatal("face-down types ignored")
	}
	if plan.Select(nil) != nil {
		t.Fatal("nil game returned members")
	}
	var zero PermanentCollectionPredicate
	if zero.Select(g) != nil {
		t.Fatal("zero query asserted members")
	}
}

type collectionQueryTrap struct{}

func (collectionQueryTrap) permanentSet() {}
func (collectionQueryTrap) resolvePermanents(*EffectContext) []*Permanent {
	panic("query executed collection")
}

func TestPermanentCollectionQueryRejectsUnknown(t *testing.T) {
	trap := MatchingPermanents(NewPermanentFilter("Land", func(*Permanent, *Game) bool { panic("query executed filter") }))
	for _, set := range []PermanentSet{nil, collectionQueryTrap{}, trap, ControlledPermanents(IsLand),
		UnionPermanents(), IntersectPermanents(), UnionPermanents(MatchingPermanents(AnyPermanent), trap),
		IntersectPermanents(MatchingPermanents(Or()), trap), UnionPermanents(trap, MatchingPermanents(IsLand)),
		MatchingPermanents(Or(AnyPermanent, NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque") }))),
	} {
		if plan, ok := InspectPermanentCollection(set); ok || plan.Select(NewGame(NewBasePlayer("A"), NewBasePlayer("B"))) != nil {
			t.Fatal("unknown collection supplied a partial query")
		}
	}
}

func TestPermanentCollectionQueryNoCallbacks(t *testing.T) {
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	g.PutOnBattlefield(NewLand("Land"), g.ActivePlayerObj().PlayerID())
	g.random.SetRandomResults([]int{3, 1, 4})
	beforeRandom := g.random.Clone()
	filter := IsLand
	filter.fn = func(*Permanent, *Game) bool { panic("query called typed filter") }
	plan, ok := InspectPermanentCollection(UnionPermanents(MatchingPermanents(filter), MatchingPermanents(filter)))
	if !ok || len(plan.Select(g)) != 1 {
		t.Fatal("query failed to deduplicate")
	}
	if !reflect.DeepEqual(g.random, beforeRandom) || g.StackSize() != 0 {
		t.Fatal("query changed game state")
	}
	for _, member := range g.zones.battlefield {
		member.GrantBaseAttr(core.Indestructible)
		g.AddRegenerationShield(member.ID())
	}
	if len(plan.Select(g)) != 1 {
		t.Fatal("selection inferred a destruction outcome")
	}

}
