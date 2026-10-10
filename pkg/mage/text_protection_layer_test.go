package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextProtectionLayerValidationIsAtomic(t *testing.T) {
	for _, scenario := range []string{"legacy", "no-builder", "no-group", "changed-kind", "changed-marker"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					option := WithAbility(ProtectionFromColors(core.White).WithLayeredColorTextRevision())
					if scenario == "legacy" || (fresh && scenario == "changed-marker") {
						option = WithAbility(ProtectionFromColors(core.White).WithColorTextRevision())
					}
					if fresh && scenario == "changed-kind" {
						option = WithAbility(StaticAbility(GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent)))
					}
					if scenario != "no-group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					return NewCreature("Source", "", 1, 1, option)
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
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			}
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(card, a.PlayerID())
			before := slices.Clone(p.RuntimeAbilities)
			attrs, printed := p.baseAttrs, p.printedAbilityAttrs
			count := len(g.effects.effects)
			change, _ := ReplaceColorWord(core.White, core.Blue)
			if err := g.AddPrintedProtectionTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err == nil {
				t.Fatal("invalid installation accepted")
			}
			current := g.FindPermanent(p.ID())
			if !slices.Equal(before, current.RuntimeAbilities) || attrs != current.baseAttrs || printed != current.printedAbilityAttrs || len(g.effects.effects) != count || len(current.layeredProtectionOrigins) != 0 {
				t.Fatal("failed installation changed state")
			}
		})
	}
}

func TestTextProtectionLayerRestoredCloneContext(t *testing.T) {
	for _, tc := range []struct {
		name       string
		suppressed bool
	}{{"active", false}, {"suppressed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			build := func() Card {
				return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(ProtectionFromColors(core.White).WithLayeredColorTextRevision())))
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
				word := core.White
				for _, change := range changes {
					word = change.ApplyColor(word)
				}
				keyword := word
				return func() Card {
					return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(ProtectionFromColors(keyword).WithLayeredColorTextRevision())))
				}, nil
			})(card.(*BaseCard))
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			change, _ := ReplaceColorWord(core.White, core.Blue)
			if err := g.AddPrintedProtectionTextEffect(p, id, core.EndOfTurn, change); err != nil {
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
func TestTextProtectionLayerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "empty", "colorless", "type", "exemption"} {
		t.Run(scenario, func(t *testing.T) {
			a := ProtectionFromColors(core.Red)
			switch scenario {
			case "empty":
				a = ProtectionFromColors()
			case "colorless":
				a = ProtectionFromColors(core.Colorless)
			case "type":
				a = ProtectionFromCardType(core.TypeCreature)
			case "exemption":
				a.ExemptCardID = uuid.New()
			}
			a.WithLayeredColorTextRevision()
			if err := ValidateCardDefinition("Declaration", NewCreature("Declaration", "", 1, 1, WithAbility(a))); (err == nil) != (scenario == "valid") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}
