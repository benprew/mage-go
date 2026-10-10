package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticActivationObjectAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "reverse", "nil-source", "empty", "invalid", "duration", "face-down", "phased", "returned", "no-builder", "no-group", "swapped", "lost-marker", "changed-condition", "changed-body", "changed-cost", "changed-target", "changed-timing", "extra-rule", "keyword", "missing-action", "nil-action", "nil-static", "missing-registration"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					effect := SourcePTWhileControllerControlsBasicLand("Swamp", 1, 1)
					if changed && scenario == "changed-condition" {
						effect = SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)
					}
					static := StaticAbility(effect).WithLayeredPTTextRevision()
					action := NewAction(ActionActivated, WithCost(ManaCostOf("B")), WithEffect(Regenerate(This)), WithLayeredTextRevisionActivationState())
					if changed {
						switch scenario {
						case "changed-body":
							WithEffect(GainLifeFor(You, Fixed(1)))(action)
						case "changed-cost":
							WithCost(GenericCost(1))(action)
						case "changed-target":
							WithTarget(TargetPermanent(IsCreature))(action)
						case "changed-timing":
							WithSorcerySpeed()(action)
						}
					}
					first, second := WithDefinitionAbilityGroup(1, WithAbility(static)), WithDefinitionAbilityGroup(2, WithAbility(action))
					if scenario == "no-group" {
						second = WithAbility(action)
					}
					if changed && scenario == "swapped" {
						first, second = WithDefinitionAbilityGroup(2, WithAbility(static)), WithDefinitionAbilityGroup(1, WithAbility(action))
					}
					if scenario == "reverse" {
						first, second = second, first
					}
					opts := []CardOption{first, second}
					if !changed || scenario != "lost-marker" {
						opts = append(opts, WithStaticActivationSpellTextContinuation())
					}
					if changed && scenario == "extra-rule" {
						opts = append(opts, WithAbility(NewAction(ActionActivated, WithEffect(Regenerate(This)))))
					}
					if changed && scenario == "keyword" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					return NewCreature("Mixed", "", 2, 2, opts...)
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
			switch scenario {
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "missing-action":
				g.MutablePermanent(p.ID()).RuntimeAbilities = p.RuntimeAbilities[:1]
			case "nil-action":
				g.MutablePermanent(p.ID()).RuntimeAbilities[1] = (*ActionDefinition)(nil)
			case "nil-static":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = (*StaticAbilityHolder)(nil)
			case "missing-registration":
				for i, e := range g.effects.effects {
					if g.printedStaticRegistration(e, p, p.RuntimeAbilities[0].AbilityID()) {
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
				changes = append(changes, TextChange{})
			}
			if scenario == "duration" {
				duration = core.EndOfCombat
			}
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			before, snapshot := len(g.effects.effects), g.FindPermanentIncludingPhased(p.ID())
			err = g.AddPrintedStaticActivationTextEffect(input, duration, changes...)
			if scenario == "valid" || scenario == "reverse" {
				if err != nil {
					t.Fatal(err)
				}
				if len(g.effects.effects) != before+2 {
					t.Fatal("missing sibling layers")
				}
				return
			}
			if err == nil {
				t.Fatal("invalid paired context accepted")
			}
			if len(g.effects.effects) != before || g.FindPermanentIncludingPhased(p.ID()) != snapshot {
				t.Fatal("failed preflight changed live state")
			}
		})
	}
}
