package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticLandTypeRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-count", "changed-layer", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "missing-registration", "builder-error", "invalid-change"} {
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
						holder.WithLandTypeTextRevision()
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
			if err := g.RevisePrintedStaticLandType(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, g.effects.effects) || !slices.Equal(abilities, p.RuntimeAbilities) {
				t.Fatal("rejection changed runtime state")
			}
		})
	}
}

func TestTextStaticLandTypeRegistrationOrder(t *testing.T) {
	build := func(subtype string) CardFactory {
		return func() Card {
			return NewEnchantment("Static", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(BecomesBasicLandsEffect(IsLand, subtype)).WithLandTypeTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build("Plains"))
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
	land := g.PutOnBattlefield(NewLand("Subject", WithSubTypes("Forest")), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticLandType(p, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 3 || g.effects.effects[0] != before || g.effects.effects[2] != after {
		t.Fatal("registration order changed")
	}
	if !g.FindPermanent(land.ID()).HasSubType("Island") || countManaAbilityColor(g.FindPermanent(land.ID()), core.White) != 0 || countManaAbilityColor(g.FindPermanent(land.ID()), core.Blue) != 1 {
		t.Fatal("old or repeated registrations remained")
	}
	for _, e := range g.effects.effects[1:2] {
		if e.SourceID() != p.ID() {
			t.Fatal("source not installed")
		}
	}
}

func TestTextStaticLandTypeDeclaration(t *testing.T) {
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
			card := NewEnchantment("Declaration", "", WithAbility(StaticAbility(effects...).WithLandTypeTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (scenario == "valid" || scenario == "all-filter") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}

func TestTextStaticLandTypeRetainsGrantedRules(t *testing.T) {
	build := func(subtype string) CardFactory {
		return func() Card {
			return NewEnchantment("Source", "", WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(BecomesBasicLandsEffect(IsLand, subtype)).WithLandTypeTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build("Plains"))
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
	printed := NewManaAbility(core.Green)
	land := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3, WithCardType(core.TypeLand), WithSubTypes("Forest", "Elf"), WithSuperTypes(core.SuperLegendary), WithAbility(printed)), a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	grant := FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		ability := NewManaAbility(core.Black)
		ability.SetSource(land.ID())
		ability.SetController(a.PlayerID())
		if p := g.MutablePermanent(land.ID()); p != nil {
			p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(ability))
		}
		return nil
	})
	grant.SetSourceID(source.ID())
	g.AddContinuousEffect(grant)
	change, _ := ReplaceBasicLandWord(core.PlainsLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticLandType(source, source.RuntimeAbilities[0].AbilityID(), change); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
		p := g.FindPermanent(land.ID())
		if !p.HasSubType("Island") || p.HasSubType("Forest") || !p.HasSubType("Elf") || !p.HasType(core.TypeCreature) || !IsLegendary.Match(p, g) || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 {
			t.Fatal("type change lost unrelated characteristics")
		}
		if hasRuntimeAbility(p, printed) || countManaAbilityColor(p, core.Green) != 0 || countManaAbilityColor(p, core.White) != 0 || countManaAbilityColor(p, core.Blue) != 1 || countManaAbilityColor(p, core.Black) != 1 {
			t.Fatal("printed, intrinsic or granted mana incorrect")
		}
	}
}
