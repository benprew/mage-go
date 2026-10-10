package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticGrantLayerDeclaration(t *testing.T) {
	for _, tc := range []string{"valid", "valid-mixed", "empty", "nil", "type", "duration", "pt"} {
		t.Run(tc, func(t *testing.T) {
			effects := []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent)}
			switch tc {
			case "valid-mixed":
				effects = append(effects, BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent))
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
			}
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithLayeredGrantTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (tc == "valid" || tc == "valid-mixed") {
				t.Fatalf("%s: %v", tc, err)
			}
		})
	}
}

func TestTextStaticGrantLayerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "legacy-marker", "changed-count", "changed-layer", "changed-order", "changed-duration", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					effects := []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent), BoostAllCreaturesIncludingSelf(1, 2, AnyPermanent)}
					if fresh && scenario == "changed-count" {
						effects = append(effects, GrantKeywordToAllIncludingSource(core.Trample, AnyPermanent))
					}
					if fresh && scenario == "changed-layer" {
						effects = []ContinuousEffect{FuncContinuousEffect(core.LayerType, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })}
					}
					if fresh && scenario == "changed-order" {
						effects[0], effects[1] = effects[1], effects[0]
					}
					if fresh && scenario == "changed-duration" {
						effects[1] = FuncContinuousEffectWithSublayer(core.LayerPT, core.SublayerPT_Modify, core.EndOfTurn, func(*Game, uuid.UUID) error { return nil })
					}
					holder := StaticAbility(effects...)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						if scenario == "legacy-marker" {
							holder.WithGrantTextRevision()
						} else {
							holder.WithLayeredGrantTextRevision()
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
			if err := g.AddPrintedStaticGrantTextEffect(input, id, core.EndOfTurn, change); err == nil {
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

func TestTextStaticGrantLayerRestoredCloneContext(t *testing.T) {
	for _, tc := range []struct {
		name       string
		suppressed bool
	}{{"active", false}, {"suppressed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			build := func() Card {
				return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent)).WithLayeredGrantTextRevision())))
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
			if err := g.AddPrintedStaticGrantTextEffect(p, id, core.EndOfTurn, change); err != nil {
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

func TestTextStaticGrantLayerRegistrationOrder(t *testing.T) {
	build := func(keyword core.Attr) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(
				GrantKeywordToAllIncludingSource(keyword, AnyPermanent), BoostAllCreaturesIncludingSelf(1, 2, AnyPermanent),
			).WithLayeredGrantTextRevision())))
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
	if err := g.AddPrintedStaticGrantTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 5 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("registration order changed")
	}
	if creature.CurrentPower(g) != 2 || creature.CurrentToughness(g) != 3 {
		t.Fatal("mixed P/T registration lost")
	}
	if g.FindPermanent(creature.ID()).HasKeyword(core.Flying) || !g.FindPermanent(creature.ID()).HasKeyword(core.Trample) {
		t.Fatal("old or repeated registrations remained")
	}
	for _, e := range g.effects.effects[1:3] {
		if e.SourceID() != p.ID() {
			t.Fatal("source not installed")
		}
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if len(g.effects.effects) != 4 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("expiry changed registration order")
	}
	if !g.FindPermanent(creature.ID()).HasKeyword(core.Flying) || g.FindPermanent(creature.ID()).HasKeyword(core.Trample) {
		t.Fatal("expiry did not restore original grant")
	}
}
