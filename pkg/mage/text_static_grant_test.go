package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticGrantDeclaration(t *testing.T) {
	for _, tc := range []string{"ability", "mixed", "empty", "nil", "type", "duration", "set", "pt-only"} {
		t.Run(tc, func(t *testing.T) {
			effects := []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent)}
			switch tc {
			case "mixed":
				effects = []ContinuousEffect{effects[0], BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)}
			case "empty":
				effects = nil
			case "nil":
				effects = []ContinuousEffect{effects[0], nil}
			case "type":
				effects = []ContinuousEffect{effects[0], FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			case "duration":
				effects = []ContinuousEffect{FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { return nil })}
			case "set":
				effects = []ContinuousEffect{effects[0], FuncContinuousEffectWithSublayer(core.LayerPT, core.SublayerPT_Set, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			case "pt-only":
				effects = []ContinuousEffect{BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)}
			}
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithGrantTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (tc == "ability" || tc == "mixed") {
				t.Fatalf("%s: %v", tc, err)
			}
		})
	}
}

func TestTextStaticGrantRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-count", "changed-layer", "changed-order", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					effects := []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent), BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)}
					if fresh && scenario == "changed-count" {
						effects = append(effects, GrantKeywordToAllIncludingSource(core.Trample, AnyPermanent))
					}
					if fresh && scenario == "changed-layer" {
						effects = []ContinuousEffect{FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
					}
					if fresh && scenario == "changed-order" {
						effects[0], effects[1] = effects[1], effects[0]
					}
					holder := StaticAbility(effects...)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						holder.WithGrantTextRevision()
					}
					option := WithAbility(holder)
					if scenario != "no-group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					return NewEnchantment("Static", "", option)
				}
			}
			plan, err := NewCardDefinitionPlan(build(false))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "no-builder" {
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) {
					if scenario == "builder-error" {
						return nil, assertionError("test builder failed")
					}
					return build(true), nil
				})(card.(*BaseCard))
			}
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			switch scenario {
			case "grant":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &grantedByEffect{Ability: p.RuntimeAbilities[0]}
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "missing-source":
				g.BouncePermanentToHand(p)
			case "unknown-id":
				id = uuid.New()
			case "missing-registration":
				g.effects.effects = nil
			}
			before := slices.Clone(g.effects.effects)
			abilities := slices.Clone(p.RuntimeAbilities)
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			change, _ := ReplaceColorWord(core.Red, core.Green)
			if scenario == "invalid-change" {
				change = TextChange{}
			}
			if err := g.RevisePrintedStaticGrant(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

func TestTextStaticGrantRegistrationOrder(t *testing.T) {
	build := func(keyword core.Attr) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(
				GrantKeywordToAllIncludingSource(keyword, AnyPermanent), BoostAllCreaturesIncludingSelf(1, 2, AnyPermanent),
			).WithGrantTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build(core.Flying))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(core.Trample), nil })(card.(*BaseCard))
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	before := FuncContinuousEffect(core.LayerAbility, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.effects.Add(before)
	p := g.PutOnBattlefield(card, a.PlayerID())
	after := FuncContinuousEffect(core.LayerAbility, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.effects.Add(after)
	creature := g.PutOnBattlefield(NewCreature("Subject", "", 1, 1), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticGrant(p, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 4 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("registration order changed")
	}
	if creature.CurrentPower(g) != 2 || creature.CurrentToughness(g) != 3 {
		t.Fatal("mixed P/T registration lost")
	}
	if creature.HasKeyword(core.Flying) || !creature.HasKeyword(core.Trample) {
		t.Fatal("old or repeated registrations remained")
	}
	for _, e := range g.effects.effects[1:3] {
		if e.SourceID() != p.ID() {
			t.Fatal("source not installed")
		}
	}
}

func TestTextStaticGrantCrossLayerContinuation(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	condition := StateConditionFunc("always", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 2, WithAbility(Static(WhileWithSharedSubjects(condition,
		GrantEachKeyword(MatchingPermanents(IsCreature), core.Flying), ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(1)),
	)).WithGrantTextRevision())), a.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Remover", "", WithAbility(StaticAbility(FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(game *Game, _ uuid.UUID) error {
		removeAllAbilitiesAtCurrentLayer(game, game.MutablePermanent(source.ID()))
		return nil
	})))), a.PlayerID())
	if current := g.FindPermanent(source.ID()); current.CurrentPower(g) != 3 || current.HasKeyword(core.Flying) {
		t.Fatalf("a later ability removal stopped the started mixed effect: power=%d, flying=%v", current.CurrentPower(g), current.HasKeyword(core.Flying))
	}
}
