package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticLandTypeLayerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "legacy-marker", "changed-count", "changed-layer", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					effects := []ContinuousEffect{BecomesBasicLandsEffect(IsLand, "Plains")}
					if fresh && scenario == "changed-count" {
						effects = append(effects, effects[0])
					}
					if fresh && scenario == "changed-layer" {
						effects = []ContinuousEffect{FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
					}
					holder := StaticAbility(effects...)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						if scenario == "legacy-marker" {
							holder.WithLandTypeTextRevision()
						} else {
							holder.WithLayeredLandTypeTextRevision()
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
			if err := g.AddPrintedStaticLandTypeTextEffect(input, id, core.EndOfTurn, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) || len(p.layeredStaticAbilityOrigins) != 0 {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

func TestTextStaticLandTypeLayerContext(t *testing.T) {
	effects := []func() ContinuousEffect{
		func() ContinuousEffect { return BecomesBasicLandsEffect(IsLand, "Swamp") },
		func() ContinuousEffect { return BecomesBasicLandAttachedEffect("Swamp") },
		BecomesChosenBasicLandTypeAttachedEffect,
	}
	for i := range effects {
		for j := range effects {
			if i == j {
				continue
			}
			t.Run(string(rune('A'+i))+string(rune('A'+j)), func(t *testing.T) {
				build := func(index int) CardFactory {
					return func() Card {
						return NewAura("Declaration", "", WithCastTarget(TargetPermanent(IsLand)), WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(effects[index]()).WithLayeredLandTypeTextRevision())))
					}
				}
				plan, err := NewCardDefinitionPlan(build(i))
				if err != nil {
					t.Fatal(err)
				}
				card, err := plan.Build()
				if err != nil {
					t.Fatal(err)
				}
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(j), nil })(card.(*BaseCard))
				a := NewBasePlayer("A")
				g := NewGame(a, NewBasePlayer("B"))
				host := g.PutOnBattlefield(NewLand("Host", WithSubTypes("Forest")), a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.Attach(p.ID(), host.ID())
				id := p.RuntimeAbilities[0].AbilityID()
				before := slices.Clone(g.effects.effects)
				abilities := slices.Clone(p.RuntimeAbilities)
				change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
				if err := g.AddPrintedStaticLandTypeTextEffect(p, id, core.EndOfTurn, change); err == nil {
					t.Fatal("changed land context accepted")
				}
				if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) || len(p.layeredStaticAbilityOrigins) != 0 {
					t.Fatal("rejection changed registrations")
				}
			})
		}
	}
}

func TestTextStaticLandTypeLayerRestoredCloneContext(t *testing.T) {
	for _, tc := range []struct {
		name       string
		suppressed bool
	}{{"active", false}, {"suppressed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			build := func() Card {
				return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(BecomesBasicLandsEffect(IsLand, "Swamp")).WithLayeredLandTypeTextRevision())))
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
			if err := g.AddPrintedStaticLandTypeTextEffect(p, id, core.EndOfTurn, change); err != nil {
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
func TestTextStaticLandTypeLayerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "empty", "nil", "typed-nil", "foreign", "duplicate", "all-filter", "no-type", "nonbasic", "multiple-types", "mixed"} {
		t.Run(scenario, func(t *testing.T) {
			effects := make([]ContinuousEffect, 1, 2)
			effects[0] = BecomesBasicLandsEffect(IsLand, "Plains")
			switch scenario {
			case "empty":
				effects = nil
			case "nil":
				effects[0] = nil
			case "typed-nil":
				effects[0] = (*basicLandSetEffect)(nil)
			case "foreign":
				effects[0] = AttachedEffect(core.LayerType, func(*Game, *Permanent, *Permanent) error { return nil })
			case "duplicate":
				effects = append(effects, effects[0])
			case "all-filter":
				effects[0] = &basicLandSetEffect{subtypes: []string{"Plains"}}
			case "no-type":
				effects[0] = &basicLandSetEffect{filter: IsLand}
			case "nonbasic":
				effects[0] = &basicLandSetEffect{filter: IsLand, subtypes: []string{"Desert"}}
			case "multiple-types":
				effects[0] = BecomesBasicLandsEffect(IsLand, "Plains", "Island")
			case "mixed":
				effects = append(effects, GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent))
			}
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithLayeredLandTypeTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (scenario == "valid" || scenario == "all-filter") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}

func TestTextStaticLandTypeLayerRegistrationOrder(t *testing.T) {
	build := func(subtype string) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(BecomesBasicLandsEffect(IsLand, subtype)).WithLayeredLandTypeTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build("Swamp"))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build("Island"), nil })(card.(*BaseCard))
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	before := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.effects.Add(before)
	p := g.PutOnBattlefield(card, a.PlayerID())
	after := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.effects.Add(after)
	subject := g.PutOnBattlefield(NewLand("Subject", WithSubTypes("Mountain")), a.PlayerID())
	change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.AddPrintedStaticLandTypeTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 4 || g.effects.effects[0] != before || g.effects.effects[2] != after {
		t.Fatal("installation changed registration order")
	}
	if current := g.FindPermanent(subject.ID()); !current.HasSubType("Island") {
		t.Fatal("replacement not installed once")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if len(g.effects.effects) != 3 || g.effects.effects[0] != before || g.effects.effects[2] != after {
		t.Fatal("expiry changed registration order")
	}
	if current := g.FindPermanent(subject.ID()); !current.HasSubType("Swamp") {
		t.Fatal("baseline not restored once")
	}
}
