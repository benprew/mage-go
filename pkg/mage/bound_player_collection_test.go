package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestBoundPlayerCollectionTarget(t *testing.T) {
	g, _, obj, _ := targetQueryFixture(t)
	target := DeclarePlayerTarget("player")
	obj.TargetSpecs = []Target{target.targetSpec()}
	obj.Targets = []uuid.UUID{g.players[1].PlayerID()}
	query, ok := InspectPlayerTarget(target)
	if !ok {
		t.Fatal("native player target rejected")
	}
	for range 2 {
		if id, known := query.Select(g, obj); !known || id != obj.Targets[0] {
			t.Fatal("wrong player identity")
		}
	}
	other := DeclarePlayerTarget("player")
	foreign, _ := InspectPlayerTarget(other)
	if _, known := foreign.Select(g, obj); known {
		t.Fatal("name replaced declaration identity")
	}
	obj.Targets[0] = uuid.New()
	if id, known := query.Select(g, obj); !known || id != uuid.Nil {
		t.Fatal("absent player not illegal")
	}
	obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
	obj.Targets = append(obj.Targets, uuid.New())
	if _, known := query.Select(g, obj); known {
		t.Fatal("duplicate declaration")
	}
	opponent := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	query, ok = InspectPlayerTarget(opponent)
	if !ok {
		t.Fatal("opponent target rejected")
	}
	obj.TargetSpecs = []Target{opponent.targetSpec()}
	obj.Targets = []uuid.UUID{obj.Controller}
	if id, known := query.Select(g, obj); !known || id != uuid.Nil {
		t.Fatal("self is an opponent")
	}
	obj.Targets[0] = g.players[1].PlayerID()
	if id, known := query.Select(g, obj); !known || id != obj.Targets[0] {
		t.Fatal("opponent missing")
	}
	var typedNil *PlayerTargetBinding
	for _, ref := range []PlayerRef{nil, typedNil, You, contextualPlayerTrap{}, &PlayerTargetBinding{}} {
		if _, ok := InspectPlayerTarget(ref); ok {
			t.Fatal("unsupported target accepted")
		}
	}
	target.spec = &OpponentTarget{BaseTarget: BaseTarget{min: 0, max: 2}}
	if _, ok := InspectPlayerTarget(target); ok {
		t.Fatal("plural target accepted")
	}
	var zero PlayerTargetPredicate
	if _, known := zero.Select(g, obj); known {
		t.Fatal("zero predicate")
	}
}

func TestBoundPlayerCollectionMembers(t *testing.T) {
	g, p, obj, _ := targetQueryFixture(t)
	target := DeclarePlayerTarget("player")
	p.Card.SetOwner(g.players[1].PlayerID())
	ctx := &EffectContext{Game: g, Controller: obj.Controller, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: g.players[1].PlayerID()}}}}
	filter := IsLand
	filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
	query, ok := InspectContextualPermanentCollection(IntersectPermanents(MatchingPermanents(filter), PermanentsOwnedBy(target, AnyPermanent)))
	if !ok {
		t.Fatal("bound collection rejected")
	}
	random := g.random.Clone()
	for range 2 {
		got, known := query.selectKnown(ctx)
		if !known || len(got) != 1 || got[0].ID != p.ID() {
			t.Fatal("owner differs from controller")
		}
		got[0] = PermanentVisit{}
	}
	if !reflect.DeepEqual(random, g.random) {
		t.Fatal("query changed RNG")
	}
	visits := query.Select(ctx)
	g.PhaseOut(p.ID())
	if len(query.Select(ctx)) != 0 {
		t.Fatal("phased subject")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if !reflect.DeepEqual(visits, query.Select(ctx)) {
		t.Fatal("phase changed visit")
	}
	g.RemoveFromBattlefield(p)
	p = g.PutOnBattlefield(p.Card, obj.Controller)
	if after := query.Select(ctx); len(after) != 1 || after[0].Incarnation == visits[0].Incarnation {
		t.Fatal("return used old visit")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	cloneCtx := *ctx
	cloneCtx.Game = clone
	if len(query.Select(&cloneCtx)) != 0 || len(query.Select(ctx)) != 1 {
		t.Fatal("clone membership leaked")
	}
	ctx.targetBindings = nil
	if _, known := query.selectKnown(ctx); known {
		t.Fatal("missing binding is not empty membership")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: nil}
	if got, known := query.selectKnown(ctx); !known || len(got) != 0 {
		t.Fatal("illegal player must supply empty membership")
	}
	for _, set := range []PermanentSet{PermanentsOwnedBy(contextualPlayerTrap{}, AnyPermanent), UnionPermanents(MatchingPermanents(AnyPermanent), PermanentsControlledBy(contextualPlayerTrap{}, AnyPermanent))} {
		if _, ok := InspectContextualPermanentCollection(set); ok {
			t.Fatal("opaque child accepted")
		}
	}
}

func TestBoundPlayerCollectionSpell(t *testing.T) {
	for _, state := range []string{"owned", "controlled", "undeclared", "foreign binding all illegal", "empty undeclared", "early proof", "illegal", "partial", "all illegal", "prefix"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			target := DeclarePlayerTarget("player")
			p.Card.SetOwner(obj.Controller)
			obj.TargetSpecs = []Target{target.targetSpec()}
			obj.Targets = []uuid.UUID{obj.Controller}
			subjects := PermanentsOwnedBy(target, IsLand)
			if state == "controlled" {
				subjects = PermanentsControlledBy(target, IsLand)
			}
			obj.Effects = []Effect{ReturnEachToHand(subjects)}
			want, known := false, true
			switch state {
			case "undeclared":
				obj.TargetSpecs = nil
				obj.Targets = nil
				known = false
			case "foreign binding all illegal":
				obj.TargetSpecs = []Target{DeclarePlayerTarget("player").targetSpec()}
				obj.Targets[0] = uuid.New()
				known = false
			case "empty undeclared":
				obj.TargetSpecs = nil
				obj.Targets = nil
				obj.Effects = []Effect{ReturnEachToHand(IntersectPermanents(MatchingPermanents(Not(AnyPermanent)), subjects))}
				known = false
			case "early proof":
				obj.TargetSpecs = nil
				obj.Targets = nil
				obj.Effects = []Effect{DestroyEach(MatchingPermanents(IsLand)), ReturnEachToHand(subjects)}
				known = false
			case "illegal":
				obj.Targets[0] = uuid.New()
			case "partial", "all illegal":
				obj.Targets[0] = uuid.New()
				obj.TargetSpecs = append(obj.TargetSpecs, land.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				obj.Effects = append(obj.Effects, Destroy(land))
				want = true
				if state == "all illegal" {
					g.PhaseOut(p.ID())
					want = false
				}
			case "prefix":
				obj.Effects = append(obj.Effects, DestroyEach(MatchingPermanents(IsLand)))
			}
			watched := []PermanentVisit{{p.ID(), p.incarnationID}}
			for range 2 {
				if got, k := g.InspectSpellDestruction(obj, watched); got != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", got, k, want, known)
				}
			}
		})
	}
}

func TestBoundPlayerCollectionCompleteBindings(t *testing.T) {
	g, p, obj, _ := targetQueryFixture(t)
	target := DeclarePlayerTarget("player")
	ctx := &EffectContext{Game: g, Controller: obj.Controller, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: p.ControllerID()}}}}
	sets := []PermanentSet{
		PermanentsControlledBy(target, IsLand),
		PermanentsOwnedBy(target, IsLand),
		PermanentsControlledByPlayers(OpponentsOf(target), AnyPermanent),
		UnionPermanents(PermanentsOwnedBy(target, IsLand), PermanentsControlledBy(target, IsLand)),
	}
	for _, set := range sets {
		predicate, ok := InspectContextualPermanentCollection(set)
		if !ok {
			t.Fatal("complete collection rejected")
		}
		var want []PermanentVisit
		for _, member := range set.resolvePermanents(ctx) {
			want = append(want, PermanentVisit{member.ID(), member.incarnationID})
		}
		if got, known := predicate.selectKnown(ctx); !known || !reflect.DeepEqual(got, want) {
			t.Fatal("native and pure membership differ")
		}
	}
	recipient, _ := InspectDamageRecipientCollection(PermanentDamageRecipients(PermanentsOwnedBy(target, IsLand)))
	ctx.targetBindings = nil
	if _, known := recipient.Select(ctx); known {
		t.Fatal("missing damage collection binding was accepted")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: nil}
	if got, known := recipient.Select(ctx); !known || len(got) != 0 {
		t.Fatal("illegal player gave recipients")
	}
}

func TestBoundPlayerCollectionOrderedMissing(t *testing.T) {
	g, p, obj, _ := targetQueryFixture(t)
	target := DeclarePlayerTarget("player")
	bound := DestroyEach(PermanentsOwnedBy(target, IsLand))
	ctx := &EffectContext{Game: g, Controller: obj.Controller}
	for _, effects := range []Effect{bound, Sequence(DestroyEach(MatchingPermanents(IsLand)), bound), Sequence(bound, DestroyEach(MatchingPermanents(IsLand)))} {
		predicate, ok := InspectOrderedDestruction(effects)
		if !ok {
			t.Fatal("complete declaration rejected")
		}
		if certain, known := predicate.Certain(ctx, []PermanentVisit{{p.ID(), p.incarnationID}}); certain || known {
			t.Fatal("missing binding supplied an ordered proof")
		}
	}
}
