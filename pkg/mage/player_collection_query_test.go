package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestPlayerCollectionQuerySelection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	sets := []PlayerSet{EachPlayer, noDamagePlayers{}, OnePlayer(You), OnePlayer(ActivePlayer), OnePlayer(target), OpponentsOf(You), OpponentsOf(ActivePlayer), OpponentsOf(target)}
	for _, set := range sets {
		predicate, ok := InspectPlayerCollection(set)
		if !ok {
			t.Fatalf("native collection rejected: %T", set)
		}
		for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			for _, active := range []int{0, 1} {
				g.SetActivePlayerIndex(active)
				for _, selected := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
					ctx := &EffectContext{Game: g, Controller: controller, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: selected}}}}
					want := set.resolvePlayers(ctx)
					rng := g.random.Clone()
					for range 2 {
						got, known := predicate.Select(ctx)
						if !known || !reflect.DeepEqual(got, want) {
							t.Fatalf("%T: got %v/%v want %v", set, got, known, want)
						}
						if len(got) > 0 {
							got[0] = uuid.Nil
						}
					}
					if !reflect.DeepEqual(rng, g.random) || a.Life() != 20 || b.Life() != 20 {
						t.Fatal("query changed state")
					}
					clone := g.Clone()
					child := *ctx
					child.Game = clone
					got, known := predicate.Select(&child)
					if !known || !reflect.DeepEqual(got, want) {
						t.Fatal("cloned selection differs")
					}
				}
			}
		}
	}
}

func TestPlayerCollectionQueryBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("same name")
	foreign := DeclarePlayerTarget("same name")
	for _, set := range []PlayerSet{OnePlayer(target), OpponentsOf(target)} {
		predicate, ok := InspectPlayerCollection(set)
		if !ok {
			t.Fatal("bound native player rejected")
		}
		for _, frame := range []map[*targetDeclaration][]capturedPermanent{
			nil, {foreign.targetDeclaration: {{id: a.PlayerID()}}},
			{target.targetDeclaration: {{id: a.PlayerID()}, {id: b.PlayerID()}}},
			{target.targetDeclaration: {{id: uuid.Nil}}},
			{target.targetDeclaration: {{id: a.PlayerID(), incarnation: uuid.New()}}},
		} {
			if _, known := predicate.Select(&EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: frame}); known {
				t.Fatal("invalid target frame accepted")
			}
		}
		for _, values := range [][]capturedPermanent{nil, {{id: uuid.New()}}} {
			got, known := predicate.Select(&EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: values}})
			if !known || len(got) != 0 {
				t.Fatal("illegal target supplied opponents or LKI")
			}
		}
	}
	var absent *opponentsRef
	var absentTarget *PlayerTargetBinding
	for _, set := range []PlayerSet{nil, absent, opponentSetTrap{}, OnePlayer(nil), OnePlayer(absentTarget), OnePlayer(contextualPlayerTrap{}), OpponentsOf(nil), OpponentsOf(contextualPlayerTrap{}), OpponentsOf(FromPlayer(BindPlayer("saved")))} {
		if _, ok := InspectPlayerCollection(set); ok {
			t.Fatalf("opaque or incomplete collection accepted: %T", set)
		}
	}
	var zero PlayerCollectionPredicate
	if _, known := zero.Select(&EffectContext{Game: g, Controller: a.PlayerID()}); known {
		t.Fatal("zero predicate accepted")
	}
	predicate, _ := InspectPlayerCollection(OpponentsOf(You))
	for _, ctx := range []*EffectContext{nil, {}, {Game: g}, {Game: g, Controller: uuid.New()}} {
		if _, known := predicate.Select(ctx); known {
			t.Fatal("invalid context accepted")
		}
	}
	native := OpponentsOf(You).(*opponentsRef)
	predicate, _ = InspectPlayerCollection(native)
	native.player = contextualPlayerTrap{}
	got, known := predicate.Select(&EffectContext{Game: g, Controller: a.PlayerID()})
	if !known || len(got) != 1 || got[0] != b.PlayerID() {
		t.Fatal("inspection retained mutable callback")
	}
}

func TestPlayerCollectionQueryDamage(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		for _, state := range []string{"only", "prefix", "zero prefix", "suffix", "unknown suffix", "unbound suffix", "all illegal", "unbound all illegal"} {
			t.Run(map[bool]string{false: "player", true: "mixed"}[mixed]+"/"+state, func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				players := OpponentsOf(You)
				if state == "unbound suffix" || state == "unbound all illegal" {
					players = OpponentsOf(DeclarePlayerTarget("missing"))
				}
				recipients := PlayerDamageRecipients(players)
				if mixed {
					recipients = DamageRecipients(players, MatchingPermanents(IsArtifact))
				}
				amount := Fixed(2)
				if state == "zero prefix" {
					amount = Fixed(0)
				}
				damage := DealDamageEach(This, recipients, amount)
				obj.Effects = []Effect{damage}
				certain, known := false, true
				switch state {
				case "prefix", "zero prefix":
					obj.Effects = []Effect{damage, Destroy(target)}
					known = false
				case "suffix":
					obj.Effects = []Effect{Destroy(target), damage}
					certain = true
				case "unknown suffix":
					obj.Effects = []Effect{Destroy(target), DealDamageEach(This, PlayerDamageRecipients(opponentSetTrap{}), amount)}
					known = false
				case "unbound suffix":
					obj.Effects = []Effect{Destroy(target), damage}
					known = false
				case "all illegal":
					g.RemoveFromBattlefield(p)
				case "unbound all illegal":
					g.RemoveFromBattlefield(p)
					known = false
				}
				for _, game := range []*Game{g, g.Clone()} {
					rng := game.random.Clone()
					for range 2 {
						got, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
						if got != certain || k != known {
							t.Fatalf("got %v/%v want %v/%v", got, k, certain, known)
						}
					}
					if !reflect.DeepEqual(game.random, rng) || game.GetPlayer(obj.Controller).Life() != 20 {
						t.Fatal("query changed game")
					}
				}
			})
		}
	}
}

func TestPlayerCollectionQueryRecipients(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	member := g.PutOnBattlefield(NewArtifact("member", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, set := range []PlayerSet{EachPlayer, noDamagePlayers{}, OnePlayer(You), OpponentsOf(You)} {
		for _, mixed := range []bool{false, true} {
			recipients := PlayerDamageRecipients(set)
			if mixed {
				recipients = DamageRecipients(set, MatchingPermanents(IsArtifact))
			}
			plan, ok := InspectDamageRecipientCollection(recipients)
			if !ok {
				t.Fatal("native recipients rejected")
			}
			for _, phase := range []bool{false, true} {
				if phase {
					g.PhaseOut(member.ID())
				}
				got, known := plan.Select(ctx)
				var want []DamageTargetSelection
				for _, id := range set.resolvePlayers(ctx) {
					want = append(want, DamageTargetSelection{Player: id})
				}
				if mixed && !phase {
					want = append(want, DamageTargetSelection{Permanent: PermanentVisit{member.ID(), member.incarnationID}})
				}
				if !known || !reflect.DeepEqual(got, want) {
					t.Fatalf("mixed=%v phase=%v: %v/%v want %v", mixed, phase, got, known, want)
				}
				if phase {
					g.PhaseIn([]uuid.UUID{member.ID()})
				}
			}
		}
	}
	for _, bad := range []DamageRecipientSet{PlayerDamageRecipients(nil), PlayerDamageRecipients(opponentSetTrap{}), DamageRecipients(OpponentsOf(You), collectionDamageSetTrap{})} {
		if _, ok := InspectDamageRecipientCollection(bad); ok {
			t.Fatal("partial recipient declaration accepted")
		}
	}
}
