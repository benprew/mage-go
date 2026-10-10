package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextAttachedObjectAtomic(t *testing.T) {
	for _, scenario := range []string{"valid-grant", "valid-action", "changed-action-cost", "changed-action-body", "changed-action-limit", "changed-grant-count", "changed-grant-kind", "lost-rule-marker", "duplicate-marker", "nil-source", "unmarked", "lost-marker", "changed-kind", "changed-group", "missing-group", "changed-target", "extra-rule", "keyword", "cost", "missing-builder", "missing-runtime", "nil-runtime", "missing-registration", "empty", "invalid", "duration", "face-down", "phased", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					var ability Ability = StaticAbility(GrantAbilityToAttached(core.Flying, core.AttachAura)).WithLayeredAbilityTextRevision()
					if scenario == "valid-action" || scenario == "changed-action-cost" || scenario == "changed-action-body" || scenario == "changed-action-limit" || changed && scenario == "changed-kind" {
						ability = NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(1))), WithLayeredTextRevisionActivationState())
					}

					if changed {
						switch scenario {
						case "changed-action-cost":
							WithCost(GenericCost(1))(ability.(*ActionDefinition))
						case "changed-action-body":
							WithEffect(GainLifeFor(You, Fixed(2)))(ability.(*ActionDefinition))
						case "changed-action-limit":
							WithOncePerTurn()(ability.(*ActionDefinition))
						case "changed-grant-count":
							ability.(*StaticAbilityHolder).Effects = append(ability.(*StaticAbilityHolder).Effects, GrantAbilityToAttached(core.FirstStrike, core.AttachAura))
						case "changed-grant-kind":
							ability = StaticAbility(GrantKeywordToAllIncludingSource(core.Flying, IsCreature)).WithLayeredAbilityTextRevision()
						case "lost-rule-marker":
							ability.(*StaticAbilityHolder).abilityTextLayer = false
						}
					}
					group := 1
					if changed && scenario == "changed-group" {
						group = 2
					}
					rule := WithDefinitionAbilityGroup(group, WithAbility(ability))
					if scenario == "missing-group" {
						rule = WithAbility(ability)
					}
					target := CreatureEnchantTarget()
					if changed && scenario == "changed-target" {
						target = TargetPermanent(IsLand)
					}
					opts := []CardOption{WithSubTypes("Aura"), WithCastTarget(target), rule}
					if scenario != "unmarked" && (!changed || scenario != "lost-marker") {
						opts = append(opts, WithAttachedObjectText())
					}
					if scenario == "duplicate-marker" {
						opts = append(opts, WithAttachedObjectText())
					}
					if scenario == "extra-rule" {
						opts = append(opts, WithDefinitionAbilityGroup(3, WithAbility(NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(1)))))))
					}
					if changed && scenario == "keyword" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					if changed && scenario == "cost" {
						opts = append(opts, WithAdditionalCost(GenericCost(1)))
					}
					return NewEnchantment("Attached", "", opts...)
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
			if scenario != "missing-builder" {
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			}
			g, a, _ := newSpellCopyGame()
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 3), a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(p.ID(), host.ID())
			g.ApplyContinuousEffects()
			switch scenario {
			case "missing-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = nil
			case "nil-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = []Ability{(*StaticAbilityHolder)(nil)}
			case "missing-registration":
				for i, e := range g.effects.effects {
					if g.printedStaticRegistration(e, p, p.RuntimeAbilities[0].AbilityID()) {
						g.effects.effects = append(g.effects.effects[:i], g.effects.effects[i+1:]...)
						break
					}
				}
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			}
			change, _ := ReplaceColorWord(core.Red, core.Blue)
			changes := []TextChange{change}
			duration := core.Indefinite
			if scenario == "empty" {
				changes = nil
			}
			if scenario == "invalid" {
				changes = append(changes, TextChange{})
			}
			if scenario == "duration" {
				duration = core.EndOfCombat
			}
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			before := len(g.effects.effects)
			snapshot := g.FindPermanentIncludingPhased(p.ID())
			err = g.AddPrintedAttachedObjectTextEffect(input, duration, changes...)
			if scenario == "valid-grant" || scenario == "valid-action" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid complete Aura context accepted")
			}
			if len(g.effects.effects) != before || g.FindPermanentIncludingPhased(p.ID()) != snapshot {
				t.Fatal("failed preflight changed live state")
			}
		})
	}
}
