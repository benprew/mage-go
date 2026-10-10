package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextLandAuraObjectAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "chosen", "no-match", "nil-source", "empty", "invalid", "duration", "face-down", "phased", "returned", "no-builder", "no-group", "changed-group", "lost-marker", "changed-target", "changed-context", "lost-choice", "changed-choice-group", "extra-rule", "lost-layer-marker", "missing-runtime", "nil-runtime", "missing-registration"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					chosen := scenario == "chosen" || scenario == "lost-choice" || scenario == "changed-choice-group"
					if changed && scenario == "changed-context" {
						chosen = true
					}
					land := "Swamp"
					if changed && scenario != "no-match" {
						land = "Island"
					}
					effect := BecomesBasicLandAttachedEffect(land)
					if chosen {
						effect = BecomesChosenBasicLandTypeAttachedEffect()
					}
					holder := StaticAbility(effect).WithLayeredLandTypeTextRevision()
					if changed && scenario == "lost-layer-marker" {
						holder.landTypeTextLayer = false
					}
					group := 1
					if changed && scenario == "changed-group" {
						group = 9
					}
					rule := WithDefinitionAbilityGroup(group, WithAbility(holder))
					if scenario == "no-group" {
						rule = WithAbility(holder)
					}
					target := LandEnchantTarget()
					if changed && scenario == "changed-target" {
						target = TargetPermanent(IsLand)
					}
					opts := []CardOption{WithCastTarget(target), rule}
					if !changed || scenario != "lost-marker" {
						opts = append(opts, WithLandTypeAuraSpellTextContinuation())
					}
					if chosen && (!changed || scenario != "lost-choice") {
						choiceGroup := 2
						if changed && scenario == "changed-choice-group" {
							choiceGroup = 8
						}
						opts = append(opts, WithDefinitionAbilityGroup(choiceGroup, WithAbility(ChooseBasicLandTypeAsEnters())))
					}
					if changed && scenario == "extra-rule" {
						opts = append(opts, WithAbility(NewActivated(TapCost(This), GainLifeFor(You, Fixed(1)))))
					}
					return NewAura("Land Aura", "", opts...)
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
			host := g.PutOnBattlefield(NewLand("Host", WithSubTypes("Forest")), a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(p.ID(), host.ID())
			g.ApplyContinuousEffects()
			switch scenario {
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "missing-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = nil
			case "nil-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = []Ability{(*StaticAbilityHolder)(nil)}
			case "missing-registration":
				for i, effect := range g.effects.effects {
					if g.printedStaticRegistration(effect, p, p.RuntimeAbilities[0].AbilityID()) {
						g.effects.effects = append(g.effects.effects[:i], g.effects.effects[i+1:]...)
						break
					}
				}
			}
			change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			changes := []TextChange{change}
			duration := core.Indefinite
			if scenario == "empty" {
				changes = nil
			}
			if scenario == "invalid" {
				changes = []TextChange{{}}
			}
			if scenario == "duration" {
				duration = core.EndOfCombat
			}
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			before, count := g.FindPermanentIncludingPhased(p.ID()), len(g.effects.effects)
			err = g.AddPrintedLandAuraObjectTextEffect(input, duration, changes...)
			valid := scenario == "valid" || scenario == "chosen" || scenario == "no-match"
			if (err == nil) != valid {
				t.Fatalf("unexpected result: %v", err)
			}
			if !valid && (g.FindPermanentIncludingPhased(p.ID()) != before || len(g.effects.effects) != count) {
				t.Fatal("rejection changed live state")
			}
			if scenario == "valid" && !g.FindPermanent(host.ID()).HasSubType("Island") {
				t.Fatal("changed land rule did not reach its host")
			}
		})
	}
}
