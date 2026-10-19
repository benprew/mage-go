package mage

import (
	"reflect"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeclarationTargetQuery(t *testing.T) {
	wall := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
	broad := DeclarePermanentTarget("broad", AnyPermanent)
	for _, ability := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			targets []Target
			blocked bool
		}{
			{"wall", []Target{wall}, true},
			{"broad", []Target{broad}, false},
			{"empty", nil, false},
			{"mixed", []Target{wall, broad}, false},
			{"zero broad", []Target{wall, &countBoundTarget{inner: broad.spec, count: 0}}, false},
			{"zero wall", []Target{wall, &countBoundTarget{inner: wall.spec, count: 0}}, true},
		} {
			t.Run(tc.name+map[bool]string{false: "/spell", true: "/ability"}[ability], func(t *testing.T) {
				g, _, obj, binding := targetQueryFixture(t)
				filter := CanTargetOnlySubtypeFilter("Wall")
				rule := CantBeTargetedBySpells(This, filter)
				if ability {
					rule = CantBeTargetedByAbilities(This, filter)
				}
				p := g.PutOnBattlefield(NewLand("Subject", WithAbility(Static(rule))), obj.Controller)
				obj.Targets[0] = p.ID()
				obj.targetIncarnations[p.ID()] = p.incarnationID
				obj.TargetZones[p.ID()] = ZoneBattlefield
				var options []any
				for _, target := range tc.targets {
					options = append(options, WithCastTarget(target))
				}
				obj.Card = NewSorcery("Source", "{R}", options...)
				obj.SourceID = obj.Card.ID()
				obj.IsAbility = ability
				if ability {
					obj.TargetSource = g.AbilityTargetSource(obj.Card, Activate(Costs(GenericCost(0)), GainLifeFor(You, Fixed(1))), tc.targets)
				}
				plan, _ := InspectPermanentTarget(binding)
				before := g.random.Clone()
				for _, game := range []*Game{g, g.Clone()} {
					got, known := plan.Select(game, game.StackPeek())
					if !known || (len(got) == 0) != tc.blocked {
						t.Fatalf("query %v known %v blocked %v", got, known, tc.blocked)
					}
					source := game.resolutionTargetSource(game.StackPeek())
					if game.FindPermanent(p.ID()).CanBeTargetedBy(source, obj.Controller, game) == tc.blocked {
						t.Fatal("native legality differs")
					}
				}
				if !reflect.DeepEqual(before, g.random) || g.StackSize() != 1 {
					t.Fatal("query changed state")
				}
			})
		}
	}
}

func TestDeclarationTargetQueryNoCallbacks(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	filter := OrCardFilter(HasCardTypeCardFilter(TypeCreature), CanTargetOnlySubtypeFilter("Wall"))
	filter.fn = func(Card) bool { panic("query ran typed filter") }
	source := g.PutOnBattlefield(NewLand("Rule", WithAbility(Static(CantBeTargetedBySpells(This, filter)))), p.ControllerID())
	obj.Targets[0] = source.ID()
	obj.targetIncarnations[source.ID()] = source.incarnationID
	obj.TargetZones[source.ID()] = ZoneBattlefield
	plan, _ := InspectPermanentTarget(binding)
	if got, known := plan.Select(g, obj); !known || len(got) != 1 {
		t.Fatal("typed declaration tree unknown")
	}
	opaque := OrCardFilter(AnyCardFilter(), NewCardFilter("can target only Wall", func(Card) bool { panic("query ran opaque filter") }))
	if opaque.characteristics != nil {
		t.Fatal("partial tree accepted")
	}
}

func TestDeclarationTargetQueryFilters(t *testing.T) {
	wall := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
	for _, declared := range [][]Target{nil, {wall}, {DeclarePermanentTarget("broad", AnyPermanent)}} {
		source := SpellTargetSourceWithDeclarations(NewInstant("Source", "{R}"), declared)
		leaf := CanTargetOnlySubtypeFilter("Wall")
		leaf.fn = func(Card) bool { panic("query ran leaf callback") }
		for _, filter := range []CardFilter{leaf, AndCardFilter(HasColorCardFilter(Red), leaf), OrCardFilter(HasCardTypeCardFilter(TypeCreature), leaf)} {
			if filter.characteristics == nil || filter.characteristics.matches(source) != (len(declared) > 0 && declared[0] == wall) {
				t.Fatal("declaration composition differs")
			}
		}
	}
	for _, filter := range []CardFilter{
		NewCardFilter("can target only Wall", func(Card) bool { panic("opaque callback") }),
		OrCardFilter(AnyCardFilter(), NewCardFilter("opaque", func(Card) bool { panic("opaque callback") })),
		AndCardFilter(CanTargetOnlySubtypeFilter("Wall"), NewCardFilter("opaque", func(Card) bool { panic("opaque callback") })),
	} {
		if filter.characteristics != nil {
			t.Fatal("opaque declaration tree accepted")
		}
	}
}
