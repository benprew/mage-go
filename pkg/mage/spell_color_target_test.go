package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellColorTarget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		filter SpellFilter
		colors []core.Color
		want   bool
	}{
		{"blue", SpellCardMatches(HasColorCardFilter(core.Blue)), []core.Color{core.Blue}, true},
		{"multicolor", SpellCardMatches(HasColorCardFilter(core.Blue)), []core.Color{core.Blue, core.Red}, true},
		{"wrong color", SpellCardMatches(HasColorCardFilter(core.Blue)), []core.Color{core.Red}, false},
		{"colorless", SpellCardMatches(HasColorCardFilter(core.Blue)), nil, false},
		{"not", NotSpellFilter(SpellCardMatches(HasColorCardFilter(core.Red))), []core.Color{core.Blue}, true},
		{"mixed tree", AllSpellFilters(SpellCardMatches(HasCardTypeCardFilter(core.TypeInstant)), AnySpellFilter(SpellCardMatches(HasColorCardFilter(core.Blue)), SpellCardMatches(HasColorCardFilter(core.Red)))), []core.Color{core.Red}, true},
		{"card tree", SpellCardMatches(AndCardFilter(HasCardTypeCardFilter(core.TypeInstant), OrCardFilter(HasColorCardFilter(core.Blue), HasColorCardFilter(core.Red)))), []core.Color{core.Red}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, victim, obj, target := counterInstructionFixture(t)
			victim.Card.(*BaseCard).SetColorOverride(tc.colors)
			target.spec.(*typedSpellTarget).contextualFilter = tc.filter
			predicate, ok := InspectSpellTarget(target)
			if !ok {
				t.Fatal("native color tree rejected")
			}
			for _, game := range []*Game{g, g.Clone()} {
				selected, known := predicate.Select(game, game.GetStack().FindByID(obj.ID))
				if !known || (selected == victim.ID) != tc.want {
					t.Fatalf("selection = %v/%v", selected, known)
				}
			}
			if tc.filter.matches(g, obj.Controller, obj.Card, victim) != tc.want {
				t.Fatal("runtime filter differs")
			}
			g.ChangeObjectColor(victim.ID, core.Green)
			selected, known := predicate.Select(g, obj)
			if !known || (selected != uuid.Nil) != tc.filter.matches(g, obj.Controller, obj.Card, victim) {
				t.Fatal("selection used old colors")
			}
		})
	}
}

func TestSpellColorTargetUnknown(t *testing.T) {
	for _, color := range []core.Color{core.Colorless, core.AnyColor, core.Color(-1), core.Color(99)} {
		target := DeclareSpellTarget("spell").WithFilter(SpellCardMatches(HasColorCardFilter(color)))
		if _, ok := InspectSpellTarget(target); ok {
			t.Fatal("invalid color accepted")
		}
	}
	g, victim, obj, target := counterInstructionFixture(t)
	victim.Card.(*BaseCard).SetColorOverride([]core.Color{core.Blue})
	native := HasColorCardFilter(core.Blue)
	native.fn = func(Card) bool { panic("native callback") }
	target.spec.(*typedSpellTarget).contextualFilter = SpellCardMatches(native)
	predicate, ok := InspectSpellTarget(target)
	if !ok {
		t.Fatal("native color rejected")
	}
	if got, known := predicate.Select(g, obj); !known || got != victim.ID {
		t.Fatal("native color plan lost")
	}
	opaque := SpellCardMatches(NewCardFilter("blue card", func(Card) bool { panic("opaque callback") }))
	target.spec.(*typedSpellTarget).contextualFilter = AnySpellFilter(SpellCardMatches(AnyCardFilter()), opaque)
	if _, ok := InspectSpellTarget(target); ok {
		t.Fatal("opaque descendant accepted")
	}
	if _, known := predicate.Select(g, obj); known {
		t.Fatal("stale predicate accepted invalid declaration")
	}
}

func TestSpellColorTargetRuntimeOverride(t *testing.T) {
	g, victim, obj, target := counterInstructionFixture(t)
	target.spec.(*typedSpellTarget).contextualFilter = SpellCardMatches(HasColorCardFilter(core.Blue))
	g.ChangeObjectColor(victim.ID, core.Blue)
	if !target.spec.(*typedSpellTarget).contextualFilter.matches(g, obj.Controller, obj.Card, victim) {
		t.Fatal("runtime filter missed stack color override")
	}
	g.ChangeObjectColor(victim.ID, core.Red)
	if target.spec.(*typedSpellTarget).contextualFilter.matches(g, obj.Controller, obj.Card, victim) {
		t.Fatal("runtime filter kept old stack color")
	}
}

func TestSpellColorTargetOverrideIsolation(t *testing.T) {
	g, victim, obj, target := counterInstructionFixture(t)
	target.spec.(*typedSpellTarget).contextualFilter = SpellCardMatches(HasColorCardFilter(core.Blue))
	g.ChangeObjectColor(victim.ID, core.Blue)
	predicate, ok := InspectSpellTarget(target)
	if !ok {
		t.Fatal("native target rejected")
	}
	clone := g.Clone()
	clone.ChangeObjectColor(victim.ID, core.Red)
	if got, known := predicate.Select(clone, clone.GetStack().FindByID(obj.ID)); !known || got != uuid.Nil {
		t.Fatal("clone color ignored")
	}
	if got, known := predicate.Select(g, obj); !known || got != victim.ID {
		t.Fatal("clone changed original colors")
	}
	if HasColorCardFilter(core.Blue).Match(victim.Card) {
		t.Fatal("stack override changed source card")
	}
	copied := g.CopyStackObjectDirect(victim, victim.Controller, false)
	if target.spec.(*typedSpellTarget).contextualFilter.matches(g, obj.Controller, obj.Card, copied) {
		t.Fatal("copy kept a color change that is not a copiable value")
	}
	g.ChangeObjectColor(copied.ID, core.Blue)
	legacy := DeclareSpellTarget("legacy", HasColorCardFilter(core.Blue))
	legacyPlan, ok := InspectSpellTarget(legacy)
	if !ok {
		t.Fatal("legacy color declaration rejected")
	}
	if len(legacyPlan.spec.Possible(obj.Controller, obj.Card, g)) != 2 {
		t.Fatal("legacy declaration missed spell colors")
	}
	g.CounterStackObject(victim.ID)
	if got, known := predicate.Select(g, obj); !known || got != uuid.Nil {
		t.Fatal("departed color target is legal")
	}
	p := g.GetPlayer(victim.Controller)
	p.RemoveFromGraveyard(victim.Card.ID())
	p.AddToHand(victim.Card)
	if err := g.CastSpellByID(p.PlayerID(), victim.Card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if got, known := predicate.Select(g, obj); !known || got != uuid.Nil {
		t.Fatal("recast restored old target")
	}
}
