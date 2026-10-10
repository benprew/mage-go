package mage

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextLandwalkLayerDeclarations(t *testing.T) {
	for _, kw := range []core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk, core.Desertwalk, core.Flying, 0} {
		card := NewCreature("Declaration", "", 1, 1, WithLayeredPrintedBasicLandwalk(kw))
		valid := slices.Contains([]core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk}, kw)
		if err := ValidateCardDefinition("Declaration", card); (err == nil) != valid {
			t.Fatalf("%v: %v", kw, err)
		}
	}
}

func TestTextLandwalkLayerValidationIsAtomic(t *testing.T) {
	for _, scenario := range []string{"legacy", "no-builder", "no-group", "changed-kind", "changed-marker"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					option := WithLayeredPrintedBasicLandwalk(core.Swampwalk)
					if scenario == "legacy" || (fresh && scenario == "changed-marker") {
						option = WithPrintedBasicLandwalk(core.Swampwalk)
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
			change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			if err := g.AddPrintedBasicLandwalkTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err == nil {
				t.Fatal("invalid installation accepted")
			}
			current := g.FindPermanent(p.ID())
			if !slices.Equal(before, current.RuntimeAbilities) || attrs != current.baseAttrs || printed != current.printedAbilityAttrs || len(g.effects.effects) != count || len(current.layeredLandwalkOrigins) != 0 {
				t.Fatal("failed installation changed state")
			}
		})
	}
}

func TestTextLandwalkLayerRestoredCloneContext(t *testing.T) {
	for _, tc := range []struct {
		name       string
		suppressed bool
	}{{"active", false}, {"suppressed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			build := func() Card {
				return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithLayeredPrintedBasicLandwalk(core.Swampwalk)))
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
				word := core.SwampLandType
				for _, change := range changes {
					word = change.ApplyBasicLand(word)
				}
				keyword := map[core.BasicLandType]core.Attr{core.SwampLandType: core.Swampwalk, core.IslandLandType: core.Islandwalk}[word]
				return func() Card {
					return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithLayeredPrintedBasicLandwalk(keyword)))
				}, nil
			})(card.(*BaseCard))
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			if err := g.AddPrintedBasicLandwalkTextEffect(p, id, core.EndOfTurn, change); err != nil {
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
