package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticAbilityDeclaration(t *testing.T) {
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
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithAbilityTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (tc == "valid") {
				t.Fatalf("%s: %v", tc, err)
			}
		})
	}
}

func TestTextStaticAbilityRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-count", "changed-layer", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
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
						holder.WithAbilityTextRevision()
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
			if err := g.RevisePrintedStaticAbility(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

func TestTextStaticAbilityRegistrationOrder(t *testing.T) {
	build := func(keyword core.Attr) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(
				GrantKeywordToAllIncludingSource(keyword, AnyPermanent), GrantKeywordToAllIncludingSource(keyword, AnyPermanent),
			).WithAbilityTextRevision())))
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
	if err := g.RevisePrintedStaticAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 4 || g.effects.effects[0] != before || g.effects.effects[3] != after {
		t.Fatal("registration order changed")
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

func TestTextStaticAbilityGrantCloneIsolation(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	host := g.PutOnBattlefield(NewCreature("Host", "", 3, 3), a.PlayerID())
	source := g.PutOnBattlefield(NewEnchantment("Source", "", WithSubTypes("Aura"), WithAbility(StaticAbility(GrantProtectionToAttached(core.White, core.AttachAura)))), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	g.ApplyContinuousEffects()
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	second := clone.PutOnBattlefield(NewEnchantment("Second", "", WithSubTypes("Aura"), WithAbility(StaticAbility(GrantProtectionToAttached(core.Blue, core.AttachAura)))), a.PlayerID())
	clone.Attach(second.ID(), host.ID())
	clone.ApplyContinuousEffects()
	original := g.FindPermanent(host.ID())
	if !original.HasProtectionFrom(NewCreature("White", "{W}", 1, 1)) || original.HasProtectionFrom(NewCreature("Blue", "{U}", 1, 1)) {
		t.Fatal("clone grant reset changed original protection")
	}
	if !clone.FindPermanent(host.ID()).HasProtectionFrom(NewCreature("Blue", "{U}", 1, 1)) {
		t.Fatal("clone did not receive new grant")
	}
}
