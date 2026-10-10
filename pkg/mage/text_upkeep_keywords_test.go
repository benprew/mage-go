package mage

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextUpkeepKeywordsAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "added", "removed", "count", "landwalk", "invalid-count", "missing-group", "changed-group"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					group := 2
					if changed && scenario == "changed-group" {
						group = 3
					}
					trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
					rule := WithDefinitionAbilityGroup(group, WithAbility(trigger))
					if scenario == "missing-group" {
						rule = WithAbility(trigger)
					}
					opts := []CardOption{WithUpkeepObjectTextRevision(), rule, WithKeyword(core.Trample)}
					if !changed || scenario != "removed" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					if changed {
						switch scenario {
						case "added":
							opts = append(opts, WithKeyword(core.Haste))
						case "count":
							opts = append(opts, WithKeyword(core.Flying))
						case "landwalk":
							opts = append(opts, WithKeyword(core.Islandwalk))
						}
					}
					card := NewCreature("Upkeep keywords", "", 2, 3, opts...)
					if changed && scenario == "invalid-count" {
						card.attrSeeds[core.Flying] = 0
					}
					return card
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
			WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			g, a, _ := newSpellCopyGame()
			p := g.PutOnBattlefield(card, a.PlayerID())
			before := slices.Clone(p.RuntimeAbilities)
			effects := len(g.effects.effects)
			change, _ := ReplaceColorWord(core.Red, core.Blue)
			err = g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change)
			if (err == nil) != (scenario == "valid") {
				t.Fatalf("preflight: %v", err)
			}
			if scenario != "valid" && (len(g.effects.effects) != effects || !slices.Equal(before, p.RuntimeAbilities)) {
				t.Fatal("rejected revision changed live rules")
			}
			if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) || !g.FindPermanent(p.ID()).HasKeyword(core.Trample) {
				t.Fatal("revision changed fixed keywords")
			}
		})
	}
}

func TestTextUpkeepKeywordsDeclarations(t *testing.T) {
	for _, keyword := range []core.Attr{core.Flying, core.FirstStrike, core.Trample, core.Vigilance, core.Haste, core.Reach, core.Banding, core.Defender, core.Fear, core.Deathtouch, core.Lifelink, core.Shroud, core.Hexproof, core.Menace, core.Indestructible, core.Islandwalk, core.AttrCanAttack} {
		trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
		card := NewCreature("Declaration", "", 2, 3, WithUpkeepObjectTextRevision(), WithAbility(trigger), WithKeyword(keyword))
		valid := keyword != core.Islandwalk && keyword != core.AttrCanAttack
		if (ValidateCardDefinition(card.Name(), card) == nil) != valid {
			t.Fatalf("keyword %v declaration", keyword)
		}
	}
}
