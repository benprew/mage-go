package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticAnimationLayerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "legacy-marker", "lost-layer-marker", "changed-count", "changed-layer", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					effects := AnimateEach(MatchingPermanents(IsLand), 1, 1).staticEffects()
					if fresh && scenario == "changed-count" {
						effects = append(effects, effects[0])
					}
					if fresh && scenario == "changed-layer" {
						effects = []ContinuousEffect{FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
					}
					holder := StaticAbility(effects...)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						if scenario == "legacy-marker" || (fresh && scenario == "lost-layer-marker") {
							holder.WithAnimationTextRevision()
						} else {
							holder.WithLayeredAnimationTextRevision()
						}
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
			if err := g.AddPrintedStaticAnimationTextEffect(input, id, core.EndOfTurn, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) || len(p.layeredStaticAbilityOrigins) != 0 {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

func TestTextStaticAnimationLayerRestoredCloneContext(t *testing.T) {
	for _, tc := range []struct {
		name       string
		suppressed bool
	}{{"active", false}, {"suppressed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			build := func() Card {
				return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(Static(AnimateEach(MatchingPermanents(IsLand), 1, 1)).WithLayeredAnimationTextRevision())))
			}
			plan, err := NewCardDefinitionPlan(build)
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) {
				return build, nil
			})(card.(*BaseCard))
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			change, _ := ReplaceColorWord(core.White, core.Blue)
			if err := g.AddPrintedStaticAnimationTextEffect(p, id, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			if tc.suppressed {
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
			}
			clone := g.Clone()
			clone.MutablePermanent(p.ID()).baseController = b.PlayerID()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if clone.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != b.PlayerID() {
				t.Fatal("original restoration changed cloned declaration context")
			}
			if g.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != a.PlayerID() {
				t.Fatal("clone changed original declaration context")
			}
			afterExpiry := g.Clone()
			afterExpiry.MutablePermanent(p.ID()).baseController = b.PlayerID()
			afterExpiry.ApplyContinuousEffects()
			if g.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != a.PlayerID() {
				t.Fatal("later clone changed the expired baseline context")
			}
			if afterExpiry.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != b.PlayerID() {
				t.Fatal("expired baseline lost the later controller")
			}
		})
	}
}
func TestTextStaticAnimationLayerRegistrationOrder(t *testing.T) {
	build := func(amount int) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(Static(AnimateEach(MatchingPermanents(IsLand), amount, amount)).WithLayeredAnimationTextRevision())))
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
	subject := g.PutOnBattlefield(NewLand("Subject", WithSubTypes("Mountain")), a.PlayerID())
	change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.AddPrintedStaticAnimationTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 5 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("installation changed registration order")
	}
	if current := g.FindPermanent(subject.ID()); current.CurrentPower(g) != 2 || current.CurrentToughness(g) != 2 {
		t.Fatal("replacement not installed once")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if len(g.effects.effects) != 4 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("expiry changed registration order")
	}
	if current := g.FindPermanent(subject.ID()); current.CurrentPower(g) != 1 || current.CurrentToughness(g) != 1 {
		t.Fatal("baseline not restored once")
	}
}

func TestTextStaticAnimationLayerDeclaration(t *testing.T) {
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
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithLayeredAnimationTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (scenario == "valid" || scenario == "colored") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}
