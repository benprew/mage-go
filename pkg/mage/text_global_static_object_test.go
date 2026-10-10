package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextGlobalStaticObjectRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-kind", "changed-keyword", "changed-subtype", "missing-runtime", "missing-registration", "face-down", "phased", "duration", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					holder := StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, HasColorFilter(core.Black))).WithLayeredPTTextRevision()
					if changed && scenario == "changed-kind" {
						holder = StaticAbility(BecomesBasicLandsEffect(IsLand, "Island")).WithLayeredLandTypeTextRevision()
					}
					rule := WithAbility(holder)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(0, rule)
					}
					opts := []CardOption{WithGlobalStaticSpellTextContinuation(), rule}
					if changed && scenario == "changed-keyword" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Shrine"))
					}
					return NewEnchantment("Static", "", opts...)
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
			g, a, _ := newSpellCopyGame()
			p := g.PutOnBattlefield(card, a.PlayerID())
			change, _ := ReplaceColorWord(core.Black, core.Blue)
			duration := core.Indefinite
			switch scenario {
			case "empty":
				before := len(g.effects.effects)
				if err := g.AddPrintedGlobalStaticTextEffect(p, duration); err == nil {
					t.Fatal("empty changes accepted")
				}
				if len(g.effects.effects) != before {
					t.Fatal("failed preflight installed an effect")
				}
				return
			case "invalid":
				change = TextChange{}
			case "countered":
				g.RemoveFromBattlefield(p)
			case "missing-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = nil
			case "missing-registration":
				g.effects.effects = nil
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "duration":
				duration = core.EndOfCombat
			}
			before := len(g.effects.effects)
			if err := g.AddPrintedGlobalStaticTextEffect(p, duration, change); err == nil {
				t.Fatal("invalid context accepted")
			}
			if len(g.effects.effects) != before {
				t.Fatal("failed preflight installed an effect")
			}
		})
	}
}

func TestTextGlobalStaticObjectBindingDeclarations(t *testing.T) {
	pair := BindColorWordPair("unbound")
	change := ColorWordSubstitution(FirstColorWord(pair), SecondColorWord(pair))
	for _, effect := range []ResolvingEffect{ChangePrintedPermanentText(This, core.Indefinite, change), ChangePrintedPermanentText(This, core.Indefinite), ChangePrintedPermanentText(This, core.EndOfCombat, change), ChangePrintedPermanentText(This, core.Indefinite, nil)} {
		card := NewCreature("Invalid", "", 2, 2, WithAbility(NewAction(ActionActivated, WithEffect(effect))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid text declaration accepted")
		}
	}
}
