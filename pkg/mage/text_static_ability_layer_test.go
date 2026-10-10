package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticAbilityLayerDeclaration(t *testing.T) {
	for _, tc := range []string{"valid", "empty", "nil", "type", "duration", "pt", "mixed"} {
		t.Run(tc, func(t *testing.T) {
			effects := []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent)}
			switch tc {
			case "empty":
				effects = nil
			case "nil":
				effects = []ContinuousEffect{nil}
			case "type":
				effects = []ContinuousEffect{FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			case "duration":
				effects = []ContinuousEffect{FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { return nil })}
			case "pt":
				effects = []ContinuousEffect{FuncContinuousEffectWithSublayer(core.LayerPT, core.SublayerPT_Set, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			case "mixed":
				effects = []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent), FuncContinuousEffect(core.LayerPT, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
			}
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithLayeredAbilityTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (tc == "valid") {
				t.Fatalf("%s: %v", tc, err)
			}
		})
	}
}

func TestTextStaticAbilityLayerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "legacy-marker", "changed-count", "changed-layer", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					effects := []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent)}
					if fresh && scenario == "changed-count" {
						effects = append(effects, GrantKeywordToAllIncludingSource(core.Trample, AnyPermanent))
					}
					if fresh && scenario == "changed-layer" {
						effects = []ContinuousEffect{FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
					}
					holder := StaticAbility(effects...)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						if scenario == "legacy-marker" {
							holder.WithAbilityTextRevision()
						} else {
							holder.WithLayeredAbilityTextRevision()
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
			if err := g.AddPrintedStaticAbilityTextEffect(input, id, core.EndOfTurn, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			current := g.FindPermanent(p.ID())
			if current == nil {
				current = p
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, current.RuntimeAbilities) || len(current.layeredStaticAbilityOrigins) != 0 {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

func TestTextStaticAbilityLayerRestoredCloneContext(t *testing.T) {
	for _, tc := range []struct {
		name       string
		suppressed bool
	}{{"active", false}, {"suppressed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			build := func() Card {
				return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent)).WithLayeredAbilityTextRevision())))
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
			if err := g.AddPrintedStaticAbilityTextEffect(p, id, core.EndOfTurn, change); err != nil {
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
