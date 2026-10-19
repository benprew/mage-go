package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerHandSizeQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("hand")
	b.AddToHand(NewInstant("First", ""))
	for _, ref := range []PlayerRef{You, ActivePlayer, target} {
		predicate, ok := InspectPlayerHandSize(HandSizeOf(ref))
		if !ok {
			t.Fatal("native amount rejected")
		}
		ctx := &EffectContext{Game: g, Controller: b.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}}
		g.SetActivePlayerIndex(1)
		for range 2 {
			if got, known := predicate.Read(ctx); !known || got != 1 {
				t.Fatalf("read %d/%v", got, known)
			}
		}
		clone := g.Clone()
		clone.GetPlayer(b.PlayerID()).AddToHand(NewInstant("Second", ""))
		child := *ctx
		child.Game = clone
		if got, known := predicate.Read(&child); !known || got != 2 {
			t.Fatal("clone hand not read")
		}
		if len(b.Hand()) != 1 || a.Life() != 20 || b.Life() != 20 {
			t.Fatal("query changed game")
		}
	}
	predicate, _ := InspectPlayerHandSize(HandSizeOf(target))
	for _, frame := range []map[*targetDeclaration][]capturedPermanent{nil, {DeclarePlayerTarget("hand").targetDeclaration: {{id: b.PlayerID()}}}, {target.targetDeclaration: {{id: a.PlayerID()}, {id: b.PlayerID()}}}, {target.targetDeclaration: {{id: uuid.Nil}}}, {target.targetDeclaration: {{id: b.PlayerID(), incarnation: uuid.New()}}}} {
		if _, known := predicate.Read(&EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: frame}); known {
			t.Fatal("invalid frame accepted")
		}
	}
	for _, values := range [][]capturedPermanent{nil, {{id: uuid.New()}}} {
		if got, known := predicate.Read(&EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: values}}); !known || got != 0 {
			t.Fatal("absent player has cards")
		}
	}
	for _, ctx := range []*EffectContext{nil, {}, {Game: g, Controller: uuid.New()}} {
		if _, known := predicate.Read(ctx); known {
			t.Fatal("invalid context accepted")
		}
	}
	if _, known := (PlayerHandSizePredicate{}).Read(&EffectContext{Game: g, Controller: a.PlayerID()}); known {
		t.Fatal("zero predicate accepted")
	}
	var missing *PlayerTargetBinding
	for _, amount := range []ResolutionValue{nil, Fixed(1), damageInspectionValueTrap{}, HandSizeOf(nil), HandSizeOf(missing), HandSizeOf(lifeInspectionPlayerTrap{}), HandSizeOf(FromPlayer(BindPlayer("loop"))), MultiplyValues(Fixed(2), HandSizeOf(You))} {
		if _, ok := InspectPlayerHandSize(amount); ok {
			t.Fatalf("unsupported amount %T", amount)
		}
	}
}

func TestPlayerHandSizeQuerySpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "empty prefix", "foreign", "missing", "duplicate", "illegal", "all illegal", "unknown suffix", "unknown all illegal", "wrong source", "opaque replacement", "missing all illegal"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			player := DeclarePlayerTarget("hand")
			obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
			obj.Targets = append(obj.Targets, obj.Controller)
			amount := HandSizeOf(player)
			want, known := false, true
			switch state {
			case "foreign":
				amount = HandSizeOf(DeclarePlayerTarget("hand"))
				known = false
			case "missing", "missing all illegal":
				obj.TargetSpecs = obj.TargetSpecs[:1]
				obj.Targets = obj.Targets[:1]
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				known = false
			case "illegal", "all illegal", "unknown all illegal":
				obj.Targets[1] = uuid.New()
			}
			if state == "all illegal" || state == "unknown all illegal" || state == "missing all illegal" {
				p.GrantBaseAttr(core.Shroud)
			}
			if state == "prefix" {
				g.GetPlayer(obj.Controller).AddToHand(NewInstant("Present", ""))
			}
			if state == "opaque replacement" {
				g.effects.AddReplacement(&proposalTrap{})
			}
			g.AddPreventionShield(obj.Controller, 2)
			random := g.random.Clone()
			before := g.Clone()
			damage := DealDamageFrom(This, player, amount)
			if state == "wrong source" {
				damage = DealDamageFrom(damageSourceQueryTrap{}, player, amount)
				known = false
			}
			obj.Effects = []Effect{damage}
			switch state {
			case "prefix", "empty prefix":
				obj.Effects = append(obj.Effects, Destroy(land))
				known = false
			case "illegal":
				obj.Effects = append(obj.Effects, Destroy(land))
				want = true
			case "suffix":
				obj.Effects = append([]Effect{Destroy(land)}, obj.Effects...)
				want = true
			case "unknown suffix", "unknown all illegal":
				obj.Effects = append(obj.Effects, destructionInspectionTrap{})
				known = false
			}
			for range 2 {
				if got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || k != known {
					t.Fatalf("result %v/%v want %v/%v", got, k, want, known)
				}
			}
			if !reflect.DeepEqual(random, g.random) || !reflect.DeepEqual(before.effects.replacements, g.effects.replacements) || g.FindPermanent(p.ID()) == nil || g.GetPlayer(obj.Controller).Life() != 20 {
				t.Fatal("query executed spell")
			}
		})
	}
}
