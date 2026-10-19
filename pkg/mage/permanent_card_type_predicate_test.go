package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentCardTypePredicate(t *testing.T) {
	types := []core.CardType{core.TypeCreature, core.TypeInstant, core.TypeSorcery, core.TypeLand, core.TypeArtifact, core.TypeEnchantment, core.TypePlaneswalker}
	for mask := 0; mask < 1<<len(types); mask++ {
		var current []core.CardType
		for i, kind := range types {
			if mask&(1<<i) != 0 {
				current = append(current, kind)
			}
		}
		permanent := NewPermanent(NewToken("Type fixture", 2, 2, current, nil), uuid.Nil)
		for _, tc := range []struct {
			filter PermanentFilter
			want   bool
		}{
			{PermanentFilter{}, true}, {AnyPermanent, true},
			{HasCardTypeFilter(core.TypeLand), mask&8 != 0},
			{Not(HasCardTypeFilter(core.TypeLand)), mask&8 == 0},
			{And(IsLand, Not(IsCreature)), mask&8 != 0 && mask&1 == 0},
			{Or(IsArtifact, And(IsCreature, IsLand)), mask&16 != 0 || mask&9 == 9},
			{And(), true}, {Or(), false},
		} {
			plan, ok := InspectPermanentCardTypeFilter(tc.filter)
			if !ok || plan.Matches(current...) != tc.want || tc.filter.Match(permanent, nil) != tc.want {
				t.Fatalf("mask %d, filter %s: supported %v, match %v, want %v", mask, tc.filter.Text(), ok, plan.Matches(current...), tc.want)
			}
		}
		for i, kind := range types {
			plan, ok := InspectPermanentCardTypeFilter(HasCardTypeFilter(kind))
			if !ok || plan.Matches(current...) != (mask&(1<<i) != 0) {
				t.Fatalf("type %v at mask %d", kind, mask)
			}
		}
	}
}

func TestPermanentCardTypePredicateRejectsCallbacks(t *testing.T) {
	trap := NewPermanentFilter("Land", func(*Permanent, *Game) bool { panic("query called an opaque filter") })
	for _, filter := range []PermanentFilter{trap, Not(trap), And(Or(), trap), Or(AnyPermanent, trap), And(IsLand, trap), Or(trap, IsLand)} {
		if _, ok := InspectPermanentCardTypeFilter(filter); ok {
			t.Fatal("opaque predicate was accepted")
		}
	}
}

func TestPermanentCardTypePredicateIsolation(t *testing.T) {
	parts := []PermanentFilter{IsLand}
	and := And(parts...)
	or := Or(parts...)
	first, ok := InspectPermanentCardTypeFilter(and)
	if !ok {
		t.Fatal("known filter rejected")
	}
	parts[0] = IsCreature
	for _, filter := range []PermanentFilter{and, or} {
		plan, ok := InspectPermanentCardTypeFilter(filter)
		if !ok || !plan.Matches(core.TypeLand) || plan.Matches(core.TypeCreature) {
			t.Fatal("caller changed the predicate")
		}
		land := NewPermanent(NewLand("Land"), uuid.Nil)
		if !filter.Match(land, nil) {
			t.Fatal("runtime filter disagrees with its plan")
		}
	}
	clonedPlan := first
	if !clonedPlan.Matches(core.TypeLand) || first.Matches(core.TypeCreature) {
		t.Fatal("copied plan changed")
	}
}

func TestPermanentCardTypePredicateDoesNotCallKnownCallbacks(t *testing.T) {
	filter := HasCardTypeFilter(core.TypeLand)
	filter.fn = func(*Permanent, *Game) bool { panic("pure query called runtime code") }
	for _, filter := range []PermanentFilter{filter, Not(filter), And(filter, AnyPermanent), Or(filter, IsCreature)} {
		plan, ok := InspectPermanentCardTypeFilter(filter)
		if !ok {
			t.Fatal("typed predicate rejected")
		}
		_ = plan.Matches(core.TypeLand)
		_ = plan.Matches(core.TypeCreature)
	}
}

func TestPermanentCardTypePredicateCurrentTypes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subject := g.PutOnBattlefield(NewArtifact("Subject", "", WithAbility(Static(While(PermanentMatchesFilter(This, IsArtifact), AddCardType(This, core.TypeCreature))))), a.PlayerID())
	filter := And(IsArtifact, IsCreature)
	plan, ok := InspectPermanentCardTypeFilter(filter)
	if !ok {
		t.Fatal("typed predicate rejected")
	}
	check := func(game *Game) {
		t.Helper()
		live := game.FindPermanentIncludingPhased(subject.ID())
		var types []core.CardType
		for _, kind := range []core.CardType{core.TypeCreature, core.TypeInstant, core.TypeSorcery, core.TypeLand, core.TypeArtifact, core.TypeEnchantment, core.TypePlaneswalker} {
			if live.HasType(kind) {
				types = append(types, kind)
			}
		}
		if plan.Matches(types...) != filter.Match(live, game) {
			t.Fatal("plan disagrees with current types")
		}
	}
	check(g)
	clone := g.Clone()
	check(clone)
	clone.FindPermanent(subject.ID()).FaceDown = true
	check(clone)
	check(g)
	g.PhaseOut(subject.ID())
	check(g)
}
