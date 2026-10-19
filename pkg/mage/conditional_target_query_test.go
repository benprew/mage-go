package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestConditionalTargetQueryHistory(t *testing.T) {
	for _, state := range []string{"quiet", "attack", "block", "not-declared-blocker", "reset", "phase", "phase-cycle", "return", "control", "removed", "host-return", "reattach"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			rule := CantBeTargetedBySpellsIf(AttachedPermanent, AnyCardFilter(), NotCondition(SourceAttackedOrBlockedThisTurnCondition()))
			card := NewAura("Rule", "", WithCastTarget(TargetLand()), WithAbility(Static(rule)))
			source := g.PutOnBattlefield(card, obj.Controller)
			g.Attach(source.ID(), p.ID())
			want := false
			switch state {
			case "attack", "reset", "phase-cycle", "return":
				g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: source.ID(), PlayerID: obj.Controller})
				want = true
			case "block", "not-declared-blocker":
				g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: source.ID(), PlayerID: obj.Controller, Flag: state == "block"})
				want = state == "block"
			}
			switch state {
			case "reset":
				g.trackers.Turn.ResetForNewTurn(p.ControllerID())
				want = false
			case "phase", "phase-cycle":
				g.PhaseOut(source.ID())
				want = true
				if state == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{source.ID()})
				}
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				source = g.PutOnBattlefield(card, obj.Controller)
				g.Attach(source.ID(), p.ID())
				want = false
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: source.ID(), ControllerID: p.ControllerID(), Duration: Indefinite})
			case "removed":
				if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilities(This)), source.ID(), obj.Controller, nil); err != nil {
					t.Fatal(err)
				}
				want = true
			case "host-return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
			case "reattach":
				other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
				g.Attach(source.ID(), other.ID())
				want = true
			}
			plan, _ := InspectPermanentTarget(binding)
			g.random.SetRandomResults([]int{3, 1, 4})
			before := g.random.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				got, known := plan.Select(game, game.StackPeek())
				if !known || (len(got) == 1) != want {
					t.Fatalf("got %v known %v; want legal %v", got, known, want)
				}
				if state != "host-return" && game.FindPermanent(p.ID()).CanBeTargetedBy(game.resolutionTargetSource(game.StackPeek()), obj.Controller, game) != want {
					t.Fatal("native legality differs")
				}
			}
			if !reflect.DeepEqual(before, g.random) || g.StackSize() != 1 {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestConditionalTargetQueryBoolean(t *testing.T) {
	for _, condition := range []StateCondition{
		IsYourTurn(), SourceAttackedThisTurnCondition(), SourceAttackedOrBlockedThisTurnCondition(),
		NotCondition(IsYourTurn()),
		AllConditions(IsYourTurn(), NotCondition(SourceAttackedThisTurnCondition())),
		AnyCondition(SourceAttackedThisTurnCondition(), NotCondition(IsYourTurn())),
	} {
		for _, attacked := range []bool{false, true} {
			g, p, obj, binding := targetQueryFixture(t)
			rule := CantBeTargetedBySpellsIf(AttachedPermanent, AnyCardFilter(), condition)
			source := g.PutOnBattlefield(NewAura("Rule", "", WithCastTarget(TargetLand()), WithAbility(Static(rule))), obj.Controller)
			g.Attach(source.ID(), p.ID())
			if attacked {
				g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: source.ID(), PlayerID: obj.Controller})
			}
			plan, _ := InspectPermanentTarget(binding)
			got, known := plan.Select(g, obj)
			want := p.CanBeTargetedBy(g.resolutionTargetSource(obj), obj.Controller, g)
			if !known || (len(got) == 1) != want {
				t.Fatalf("%T: got %v known %v; want legal %v", condition, got, known, want)
			}
			g.SetActivePlayerIndex(1)
			got, known = plan.Select(g, obj)
			want = p.CanBeTargetedBy(g.resolutionTargetSource(obj), obj.Controller, g)
			if !known || (len(got) == 1) != want {
				t.Fatal("turn change did not recheck condition")
			}
		}
	}
}

func TestConditionalTargetQueryOpaque(t *testing.T) {
	trap := StateConditionFunc("opaque", nil, func(StateReader, StateContext) bool { panic("condition callback") })
	var nilPredicate *statePredicate
	for _, condition := range []StateCondition{nil, nilPredicate, trap, NotCondition(trap), AllConditions(IsYourTurn(), trap), AnyCondition(IsYourTurn(), trap), AllConditions(), AnyCondition(), NotCondition(nil), &statePredicate{kind: "not"}, &statePredicate{kind: "not", children: []StateCondition{IsYourTurn(), IsYourTurn()}}} {
		g, p, obj, binding := targetQueryFixture(t)
		rule := &spellTargetRestriction{funcContinuousEffect: FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(*Game, uuid.UUID) error { panic("apply callback") }).(*funcContinuousEffect), subject: AttachedPermanent, filter: AnyCardFilter(), condition: condition, conditional: true}
		source := g.PutOnBattlefield(NewAura("Rule", "", WithCastTarget(TargetLand())), obj.Controller)
		g.Attach(source.ID(), p.ID())
		rule.SetSourceID(source.ID())
		g.effects.effects = append(g.effects.effects, rule)
		plan, _ := InspectPermanentTarget(binding)
		if _, known := plan.Select(g, obj); known {
			t.Fatalf("accepted %T", condition)
		}
	}
}

func TestConditionalTargetQueryCloneIsolation(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	rule := CantBeTargetedBySpellsIf(AttachedPermanent, AnyCardFilter(), NotCondition(SourceAttackedOrBlockedThisTurnCondition()))
	source := g.PutOnBattlefield(NewAura("Rule", "", WithCastTarget(TargetLand()), WithAbility(Static(rule))), obj.Controller)
	g.Attach(source.ID(), p.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: source.ID(), PlayerID: obj.Controller})
	clone := g.Clone()
	clone.trackers.Turn.ResetForNewTurn(p.ControllerID())
	plan, _ := InspectPermanentTarget(binding)
	if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
		t.Fatal("clone retained old turn history")
	}
	copied := g.CopyStackObjectDirect(obj, obj.Controller, false)
	for _, object := range []*StackObject{obj, copied} {
		if got, known := plan.Select(g, object); !known || len(got) != 1 {
			t.Fatal("clone reset changed original or copied declaration")
		}
	}
}
