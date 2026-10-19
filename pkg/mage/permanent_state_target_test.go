package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentStateTargetPredicates(t *testing.T) {
	for _, combat := range []string{"none", "attacking", "blocking"} {
		for _, tapped := range []bool{false, true} {
			g, p, obj, _ := targetQueryFixture(t)
			p.Tapped = tapped
			switch combat {
			case "attacking":
				g.combat.AddAttacker(p.ID(), obj.Controller)
			case "blocking":
				other := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 1), obj.Controller)
				g.combat.AddAttacker(other.ID(), p.ControllerID())
				g.combat.AddBlocker(p.ID(), other.ID())
			}
			for _, tc := range []struct {
				filter PermanentFilter
				want   bool
			}{
				{IsTapped, tapped}, {IsUntapped, !tapped},
				{IsAttacking, combat == "attacking"}, {Not(IsAttacking), combat != "attacking"},
				{IsBlocking, combat == "blocking"},
				{And(IsLand, Or(IsTapped, IsBlocking)), tapped || combat == "blocking"},
				{Not(Or(IsTapped, IsAttacking)), !tapped && combat != "attacking"},
			} {
				binding := DeclarePermanentTarget("state", tc.filter)
				obj.TargetSpecs[0] = binding.targetSpec()
				plan, ok := InspectPermanentTarget(binding)
				if !ok {
					t.Fatalf("missing %s declaration", tc.filter.Text())
				}
				if got := binding.Possible(obj.Controller, obj.Card, g); slices.Contains(got, p.ID()) != tc.want {
					t.Fatal("native state differs")
				}
				binding.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("query ran callback") }
				for range 2 {
					got, known := plan.Select(g, obj)
					if !known || (len(got) == 1) != tc.want {
						t.Fatalf("%s/%v %s: %v/%v", combat, tapped, tc.filter.Text(), got, known)
					}
				}
			}
		}
	}
}

func TestPermanentStateTargetIdentity(t *testing.T) {
	for _, predicate := range []string{"tapped", "attacking", "blocking"} {
		for _, state := range []string{"legal", "untap", "remove", "phase", "phase cycle", "return", "control", "copy", "clone", "end combat", "no combat", "attacker leaves"} {
			t.Run(predicate+"/"+state, func(t *testing.T) {
				g, p, obj, _ := targetQueryFixture(t)
				g.TapPermanent(p)
				filter := IsTapped
				attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 1), obj.Controller)
				if predicate == "attacking" {
					filter = IsAttacking
					g.combat.AddAttacker(p.ID(), obj.Controller)
				}
				if predicate == "blocking" {
					filter = IsBlocking
					g.combat.AddAttacker(attacker.ID(), p.ControllerID())
					g.combat.AddBlocker(p.ID(), attacker.ID())
				}
				binding := DeclarePermanentTarget("state", filter)
				obj.TargetSpecs[0] = binding.targetSpec()
				obj.Effects = []Effect{Destroy(binding)}
				plan, ok := InspectPermanentTarget(binding)
				if !ok {
					t.Fatal("state target rejected")
				}
				want := true
				switch state {
				case "untap":
					g.UntapPermanent(p)
					want = predicate != "tapped"
				case "remove":
					g.RemoveFromCombat(p.ID())
					want = predicate == "tapped"
				case "phase", "phase cycle":
					g.PhaseOut(p.ID())
					want = false
					if state == "phase cycle" {
						g.PhaseIn([]uuid.UUID{p.ID()})
						want = predicate == "tapped"
					}
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(p.Card, p.ControllerID())
					want = false
				case "control":
					g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: obj.Controller, Duration: core.Indefinite})
					want = predicate == "tapped"
				case "copy":
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				case "clone":
					clone := g.Clone()
					clone.UntapPermanent(clone.FindPermanent(p.ID()))
					clone.RemoveFromCombat(p.ID())
					if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
						t.Fatal("clone ignored state change")
					}
				case "end combat":
					g.combat.Reset()
					want = predicate == "tapped"
				case "no combat":
					g.combat = nil
					want = predicate == "tapped"
				case "attacker leaves":
					g.RemoveFromBattlefield(attacker)
				}
				g.ApplyContinuousEffects()
				if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
					t.Fatalf("state selection %v/%v want %v", got, known, want)
				}
				if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); !known || certain != want {
					t.Fatalf("destruction %v/%v want %v", certain, known, want)
				}
				if state != "no combat" {
					g.ResolveStack()
					if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
						t.Fatal("native resolution differs")
					}
				}
			})
		}
	}
}

func TestPermanentStateTargetUnknown(t *testing.T) {
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque callback") })
	for _, filter := range []PermanentFilter{Or(AnyPermanent, opaque), And(IsTapped, opaque), Not(Or(IsBlocking, opaque)), AttackedThisTurn} {
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("state", filter)); ok {
			t.Fatal("partial declaration accepted")
		}
	}
	for _, option := range []PermanentTargetOption{TargetChosenByOpponent()} {
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("state", IsTapped, option)); ok {
			t.Fatal("extra restriction accepted")
		}
	}
}

func TestPermanentStateTargetCollections(t *testing.T) {
	g, p, obj, _ := targetQueryFixture(t)
	for _, filter := range []PermanentFilter{IsTapped, IsUntapped, And(IsLand, Not(IsTapped)), Or(IsTapped, IsAttacking)} {
		set := MatchingPermanents(filter)
		plan, ok := InspectContextualPermanentCollection(set)
		if !ok {
			t.Fatal("shared state predicate lost")
		}
		for _, tapped := range []bool{false, true} {
			g.MutablePermanent(p.ID()).Tapped = tapped
			ctx := &EffectContext{Game: g, Controller: obj.Controller}
			got, want := plan.Select(ctx), set.resolvePermanents(ctx)
			if len(got) != len(want) {
				t.Fatal("collection state query differs")
			}
			for i, member := range want {
				if got[i] != (PermanentVisit{member.ID(), member.incarnationID}) {
					t.Fatal("wrong collection visit")
				}
			}
		}
	}
}
