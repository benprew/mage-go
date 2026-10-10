package mage

import (
	"maps"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextUpkeepSpellKeywordsAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "added", "removed", "count", "landwalk", "invalid-count", "missing-group", "changed-group", "keyword-group"} {
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
					keywordGroup := 0
					if changed && scenario == "keyword-group" {
						keywordGroup = 4
					}
					opts := []CardOption{WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), rule, WithDefinitionAbilityGroup(keywordGroup, WithKeyword(core.Trample))}
					if !changed || scenario != "removed" {
						opts = append(opts, WithDefinitionAbilityGroup(1, WithKeyword(core.Flying)), WithDefinitionAbilityGroup(5, WithKeyword(core.Flying)))
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
			g.SetStep(core.PrecombatMain)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			seeds := maps.Clone(before.spellTextDefinition.(*BaseCard).attrSeeds)
			change, _ := ReplaceColorWord(core.Red, core.Blue)
			err = g.RevisePrintedSpellText(before.ID, change)
			if (err == nil) != (scenario == "valid") {
				t.Fatalf("preflight: %v", err)
			}
			if scenario != "valid" && g.StackPeek() != before {
				t.Fatal("rejected revision changed the stack visit")
			}
			if !maps.Equal(seeds, g.StackPeek().spellTextDefinition.(*BaseCard).attrSeeds) {
				t.Fatal("revision changed fixed keyword counts")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || !p.HasKeyword(core.Flying) || !p.HasKeyword(core.Trample) || !maps.Equal(seeds, p.Card.(*BaseCard).attrSeeds) {
				t.Fatal("entry lost fixed keywords")
			}
		})
	}
}

func TestTextUpkeepSpellKeywordsDeclarations(t *testing.T) {
	for _, keyword := range []core.Attr{core.Flying, core.FirstStrike, core.Trample, core.Vigilance, core.Haste, core.Reach, core.Banding, core.Defender, core.Fear, core.Deathtouch, core.Lifelink, core.Shroud, core.Hexproof, core.Menace, core.Indestructible, core.Islandwalk, core.AttrCanAttack} {
		trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
		card := NewCreature("Declaration", "", 2, 3, WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), WithAbility(trigger), WithKeyword(keyword))
		valid := keyword != core.Islandwalk && keyword != core.AttrCanAttack
		if (ValidateCardDefinition(card.Name(), card) == nil) != valid {
			t.Fatalf("keyword %v declaration", keyword)
		}
	}
}
