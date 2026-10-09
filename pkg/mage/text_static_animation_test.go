package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticAnimationRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-count", "changed-layer", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					effects := AnimateEach(MatchingPermanents(AnyPermanent), 1, 1).staticEffects()
					if fresh && scenario == "changed-count" {
						effects = append(effects, effects[0])
					}
					if fresh && scenario == "changed-layer" {
						effects = []ContinuousEffect{FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
					}
					holder := StaticAbility(effects...)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						holder.WithAnimationTextRevision()
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
			if err := g.RevisePrintedStaticAnimation(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

func TestTextStaticAnimationRegistrationOrder(t *testing.T) {
	build := func(amount int) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(Static(AnimateEach(MatchingPermanents(AnyPermanent), amount, amount)).WithAnimationTextRevision())))
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
	if err := g.RevisePrintedStaticAnimation(p, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 4 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("registration order changed")
	}
	if creature.CurrentPower(g) != 2 || creature.CurrentToughness(g) != 2 {
		t.Fatal("old or repeated registrations remained")
	}
	for _, e := range g.effects.effects[1:3] {
		if e.SourceID() != p.ID() {
			t.Fatal("source not installed")
		}
	}
}

func TestTextStaticAnimationDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "colored", "empty", "nil", "foreign", "partial", "order", "mixed-rule", "duplicate"} {
		t.Run(scenario, func(t *testing.T) {
			effects := AnimateEach(MatchingPermanents(AnyPermanent), 1, 1).staticEffects()
			switch scenario {
			case "colored":
				effects = AnimateEach(MatchingPermanents(AnyPermanent), 1, 1, core.Black).staticEffects()
			case "empty":
				effects = nil
			case "nil":
				effects[0] = nil
			case "foreign":
				effects[0] = FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })
			case "partial":
				effects = effects[:1]
			case "order":
				effects[0], effects[1] = effects[1], effects[0]
			case "mixed-rule":
				effects[1] = AnimateEach(MatchingPermanents(AnyPermanent), 2, 2).staticEffects()[1]
			case "duplicate":
				effects = append(effects, effects[1])
			}
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithAnimationTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (scenario == "valid" || scenario == "colored") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}
