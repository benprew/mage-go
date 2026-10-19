package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestGlobalTargetQueryLifecycle(t *testing.T) {
	for _, state := range []string{"active", "unmatched", "unrelated", "phase", "phase-cycle", "departed", "returned", "removed", "host-return", "reattach", "control"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			rule := CantBeTargetedBySpells(AttachedPermanent, HasColorCardFilter(Red))
			card := NewAura("Restriction", "", WithCastTarget(TargetLand()), WithAbility(Static(rule)))
			source := g.PutOnBattlefield(card, obj.Controller)
			g.Attach(source.ID(), p.ID())
			want := false
			switch state {
			case "unmatched":
				obj.ColorOverride = new([]Color)
				*obj.ColorOverride = []Color{Green}
				want = true
			case "unrelated", "reattach":
				other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
				g.Attach(source.ID(), other.ID())
				want = true
			case "phase", "phase-cycle":
				g.PhaseOut(source.ID())
				want = true
				if state == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{source.ID()})
					want = false
				}
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "returned" {
					g.PutOnBattlefield(card, obj.Controller)
				}
				want = true
			case "removed":
				if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilities(This)), source.ID(), obj.Controller, nil); err != nil {
					t.Fatal(err)
				}
				want = true
			case "host-return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ControllerID(), TargetID: source.ID(), ControllerID: p.ControllerID(), Duration: Indefinite})
			}
			plan, _ := InspectPermanentTarget(binding)
			before := g.random.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				got, known := plan.Select(game, game.StackPeek())
				if !known || (len(got) == 1) != want {
					t.Fatalf("got %v, known %v; want legal %v", got, known, want)
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

func TestGlobalTargetQueryNoCallbacks(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	filter := HasColorCardFilter(Red)
	filter.fn = func(Card) bool { panic("typed filter callback") }
	rule := CantBeTargetedBySpells(This, filter)
	source := g.PutOnBattlefield(NewLand("Subject", WithAbility(Static(rule))), p.ControllerID())
	obj.Targets = []uuid.UUID{source.ID()}
	obj.targetIncarnations[source.ID()] = source.incarnationID
	obj.TargetZones[source.ID()] = ZoneBattlefield
	plan, _ := InspectPermanentTarget(binding)
	g.effects.effects = append(g.effects.effects, FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(*Game, uuid.UUID) error { panic("unrelated apply callback") }, func(*Game, uuid.UUID) bool { panic("unrelated activity callback") }))
	if got, known := plan.Select(g, obj); !known || len(got) != 0 {
		t.Fatal("typed rule not inspected")
	}
	for _, effect := range g.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		if restriction, ok := inner.(*spellTargetRestriction); ok {
			restriction.filter = AndCardFilter(filter, NewCardFilter("opaque", func(Card) bool { panic("opaque filter callback") }))
			if _, known := plan.Select(g, obj); known {
				t.Fatal("partial filter accepted")
			}
			restriction.filter = filter
			restriction.conditional = true
			restriction.condition = StateConditionFunc("trap", nil, func(StateReader, StateContext) bool { panic("condition callback") })
			if _, known := plan.Select(g, obj); known {
				t.Fatal("conditional rule accepted")
			}
			restriction.conditional = false
			restriction.active = func(*Game, uuid.UUID) bool { panic("activity callback") }
			if _, known := plan.Select(g, obj); known {
				t.Fatal("opaque activity accepted")
			}
		}
	}
}

func TestGlobalTargetQueryAbilitySource(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "copy", "clone", "color-change", "face-down"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePermanentTarget("subject", HasCardTypeFilter(TypeLand))
			card := NewArtifact("Source", "{R}", WithAbility(Activate(Costs(ManaPayment("{0}")), Destroy(target), WithTargets(target))))
			card.SetOwner(a.PlayerID())
			actor := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewLand("Subject"), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), actor.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			g.ChangeObjectColor(actor.ID(), Blue)
			filter := AndCardFilter(HasCardTypeCardFilter(TypeArtifact), OrCardFilter(HasColorCardFilter(Blue), HasSubTypeCardFilter("Goblin")))
			rule := g.PutOnBattlefield(NewAura("Restriction", "", WithCastTarget(TargetLand()), WithAbility(Static(CantBeTargetedByAbilities(AttachedPermanent, filter)))), b.PlayerID())
			g.Attach(rule.ID(), victim.ID())
			want := false
			switch state {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(actor.ID()))
				if state == "returned" {
					g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
				}
			case "phased":
				g.PhaseOut(actor.ID())
			case "copy":
				obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "color-change":
				g.ChangeObjectColor(actor.ID(), Green)
				want = true
			case "face-down":
				g.FindPermanent(actor.ID()).FaceDown = true
				want = true
			}
			plan, _ := InspectPermanentTarget(target)
			got, known := plan.Select(g, obj)
			if !known || (len(got) == 1) != want {
				t.Fatalf("got %v known %v want legal %v", got, known, want)
			}
			g.ResolveStack()
			if (g.FindPermanent(victim.ID()) == nil) != want {
				t.Fatal("native resolution differs")
			}
		})
	}
}

func TestGlobalTargetQueryFilterComposition(t *testing.T) {
	red := NewInstant("Red", "{R}")
	artifact := NewArtifact("Artifact", "{U}")
	filters := []CardFilter{HasColorCardFilter(Red), HasCardTypeCardFilter(TypeArtifact)}
	and := AndCardFilter(filters...)
	or := OrCardFilter(filters...)
	filters[0] = AnyCardFilter()
	for _, tc := range []struct {
		filter CardFilter
		card   Card
		want   bool
	}{{and, red, false}, {and, artifact, false}, {or, red, true}, {or, artifact, true}} {
		if tc.filter.Match(tc.card) != tc.want || tc.filter.characteristics.matches(tc.card) != tc.want {
			t.Fatal("native and retained predicates differ")
		}
	}
}
