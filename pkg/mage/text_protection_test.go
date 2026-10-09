package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextProtectionDeclaration(t *testing.T) {
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
			a.WithColorTextRevision()
			if err := ValidateCardDefinition("Declaration", NewCreature("Declaration", "", 1, 1, WithAbility(a))); (err == nil) != (scenario == "valid") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}

func TestTextProtectionRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "no-builder", "no-group", "unknown-id", "nil-source", "missing-source", "invalid-change", "changed-kind"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					protection := ProtectionFromColors(core.Red)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						protection.WithColorTextRevision()
					}
					var ability Ability = protection
					if fresh && scenario == "changed-kind" {
						ability = StaticAbility(GrantKeywordToAllIncludingSource(core.Flying, AnyPermanent))
					}
					option := WithAbility(ability)
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
			change, _ := ReplaceColorWord(core.Red, core.Blue)
			if scenario == "invalid-change" {
				change = TextChange{}
			}
			if err := g.RevisePrintedProtection(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if !slices.Equal(before, p.RuntimeAbilities) || len(p.revisedPrintedProtections) != 0 {
				t.Fatal("rejection changed state")
			}
		})
	}
}
