package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticPTDeclaration(t *testing.T) {
	for _, tc := range []string{"valid", "empty", "nil", "type", "duration", "set", "mixed"} {
		t.Run(tc, func(t *testing.T) {
			effects := []ContinuousEffect{BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)}
			switch tc {
			case "empty":
				effects = nil
			case "nil":
				effects = []ContinuousEffect{nil}
			case "type":
				effects = []ContinuousEffect{FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			case "duration":
				effects = []ContinuousEffect{FuncContinuousEffect(core.LayerPT, core.EndOfTurn, func(*Game, uuid.UUID) error { return nil })}
			case "set":
				effects = []ContinuousEffect{FuncContinuousEffectWithSublayer(core.LayerPT, core.SublayerPT_Set, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			case "mixed":
				effects = []ContinuousEffect{BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent), FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			}
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithPTTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (tc == "valid") {
				t.Fatalf("%s: %v", tc, err)
			}
		})
	}
}

func TestTextStaticPTRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-count", "changed-layer", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					effects := []ContinuousEffect{BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)}
					if fresh && scenario == "changed-count" {
						effects = append(effects, BoostAllCreaturesIncludingSelf(2, 2, AnyPermanent))
					}
					if fresh && scenario == "changed-layer" {
						effects = []ContinuousEffect{FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
					}
					holder := StaticAbility(effects...)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						holder.WithPTTextRevision()
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
			if err := g.RevisePrintedStaticPTAbility(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }

func TestTextStaticPTRegistrationOrder(t *testing.T) {
	build := func(amount int) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(
				BoostAllCreaturesIncludingSelf(amount, amount, AnyPermanent), BoostAllCreaturesIncludingSelf(amount, amount, AnyPermanent),
			).WithPTTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build(1))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(2), nil })(card.(*BaseCard))
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	before := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.effects.Add(before)
	p := g.PutOnBattlefield(card, a.PlayerID())
	after := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.effects.Add(after)
	creature := g.PutOnBattlefield(NewCreature("Subject", "", 1, 1), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 4 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("registration order changed")
	}
	if creature.CurrentPower(g) != 5 || creature.CurrentToughness(g) != 5 {
		t.Fatal("old or repeated registrations remained")
	}
	for _, e := range g.effects.effects[1:3] {
		if e.SourceID() != p.ID() {
			t.Fatal("source not installed")
		}
	}
}
