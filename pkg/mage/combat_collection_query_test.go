package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatCollectionQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "{R}", 2, 4), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "{W}", 2, 4), b.PlayerID())
	g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
	sets := []PermanentSet{
		MatchingPermanents(IsAttacking), MatchingPermanents(IsBlocking), MatchingPermanents(Not(IsAttacking)),
		MatchingPermanents(And(IsCreature, Or(IsAttacking, IsBlocking))),
		MatchingPermanents(Not(Or(IsAttacking, IsBlocking))),
		PermanentsControlledBy(You, And(IsCreature, IsAttacking)),
		PermanentsControlledByPlayers(OpponentsOf(You), IsBlocking),
		UnionPermanents(MatchingPermanents(IsAttacking), MatchingPermanents(IsBlocking), MatchingPermanents(IsAttacking)),
	}
	plans := make([]ContextualPermanentCollectionPredicate, len(sets))
	for i, set := range sets {
		var ok bool
		plans[i], ok = InspectContextualPermanentCollection(set)
		if !ok {
			t.Fatalf("missing current-combat declaration %d", i)
		}
		if _, ok := InspectPermanentCollection(set); ok {
			t.Fatal("context-free inspection accepted combat")
		}
	}
	check := func(game *Game) {
		t.Helper()
		for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			ctx := &EffectContext{Game: game, Controller: controller}
			for i, set := range sets {
				var want []PermanentVisit
				for _, p := range set.resolvePermanents(ctx) {
					want = append(want, PermanentVisit{p.ID(), p.incarnationID})
				}
				if got := plans[i].Select(ctx); !reflect.DeepEqual(got, want) {
					t.Fatalf("set %d: got %v, want %v", i, got, want)
				}
			}
		}
	}
	check(g)
	clone := g.Clone()
	clone.RemoveFromCombat(attacker.ID())
	check(clone)
	if len(plans[0].Select(&EffectContext{Game: g, Controller: a.PlayerID()})) != 1 {
		t.Fatal("clone changed original combat")
	}
	g.PhaseIn(g.PhaseOut(blocker.ID()))
	check(g)
	if len(plans[1].Select(&EffectContext{Game: g, Controller: a.PlayerID()})) != 0 {
		t.Fatal("phasing restored blocking")
	}
	g.TapPermanent(attacker)
	check(g)
	if len(plans[0].Select(&EffectContext{Game: g, Controller: a.PlayerID()})) != 1 {
		t.Fatal("tapping removed attacker")
	}
	g.AddControlEffect(ControlEffectSpec{TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(g)
	if len(plans[0].Select(&EffectContext{Game: g, Controller: a.PlayerID()})) != 0 {
		t.Fatal("control change kept attacker")
	}
	g.RemoveFromBattlefield(attacker)
	returned := g.PutOnBattlefield(attacker.Card, a.PlayerID())
	check(g)
	if returned.incarnationID == attacker.incarnationID || len(plans[0].Select(&EffectContext{Game: g, Controller: a.PlayerID()})) != 0 {
		t.Fatal("returned visit inherited combat")
	}
	g.combat.Reset()
	check(g)
}

func TestCombatCollectionQueryNoCallbacks(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 4), a.PlayerID())
	g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	filter := IsAttacking
	filter.fn = func(*Permanent, *Game) bool { panic("combat filter callback") }
	inputs := []PermanentFilter{filter, IsBlocking}
	set := MatchingPermanents(And(IsCreature, Or(inputs...)))
	plan, ok := InspectContextualPermanentCollection(set)
	if !ok {
		t.Fatal("missing combat predicate")
	}
	inputs[0] = Not(IsAttacking)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	random := g.random.Clone()
	for range 3 {
		got := plan.Select(ctx)
		if len(got) != 1 || got[0].ID != p.ID() {
			t.Fatal("lost immutable declaration")
		}
		got[0] = PermanentVisit{}
	}
	if !reflect.DeepEqual(random, g.random) || g.StackSize() != 0 || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("query changed game")
	}
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque callback") })
	for _, invalid := range []PermanentSet{
		MatchingPermanents(Or(AnyPermanent, opaque)),
		MatchingPermanents(And(Or(), IsAttacking, opaque)),
		MatchingPermanents(Not(Or(IsAttacking, opaque))),
		UnionPermanents(MatchingPermanents(IsAttacking), collectionQueryTrap{}),
		PermanentsControlledBy(contextualPlayerTrap{}, IsAttacking),
	} {
		if _, ok := InspectContextualPermanentCollection(invalid); ok {
			t.Fatal("unknown descendant supplied partial query")
		}
	}
	if plan.Select(nil) != nil || plan.Select(&EffectContext{Game: g}) != nil {
		t.Fatal("invalid context supplied members")
	}
	g.combat = nil
	if len(plan.Select(ctx)) != 0 {
		t.Fatal("absent combat supplied an attacker")
	}
}
