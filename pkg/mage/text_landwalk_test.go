package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextLandwalkDeclaration(t *testing.T) {
	for _, kw := range []core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk, core.Desertwalk, core.LegendaryLandwalk, core.Flying, 0} {
		card := NewCreature("Declaration", "", 1, 1, WithPrintedBasicLandwalk(kw))
		valid := slices.Contains([]core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk}, kw)
		if err := ValidateCardDefinition("Declaration", card); (err == nil) != valid {
			t.Fatalf("%v: %v", kw, err)
		}
	}
}

func TestTextLandwalkRejects(t *testing.T) {
	for _, scenario := range []string{"grant", "face-down", "missing-marker", "no-builder", "no-group", "unknown-id", "nil-source", "missing-source", "invalid-change", "changed-kind"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					option := WithPrintedBasicLandwalk(core.Swampwalk)
					if fresh && scenario == "missing-marker" {
						option = WithKeyword(core.Swampwalk)
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
			card := build(false)()
			if scenario != "no-builder" {
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			}
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			switch scenario {
			case "grant":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = WrapGrantedAbility(p.RuntimeAbilities[0])
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "unknown-id":
				id = uuid.New()
			case "missing-source":
				g.BouncePermanentToHand(p)
			}
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			before := slices.Clone(p.RuntimeAbilities)
			attrs, printed := p.baseAttrs, p.printedAbilityAttrs
			change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			if scenario == "invalid-change" {
				change = TextChange{}
			}
			if err := g.RevisePrintedBasicLandwalk(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, p.RuntimeAbilities) || len(p.revisedPrintedLandwalks) != 0 || p.baseAttrs != attrs || p.printedAbilityAttrs != printed {
				t.Fatal("rejection changed state")
			}
		})
	}
}
