package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextActivatedObjectAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "missing-second-runtime", "changed-kind", "changed-timing", "changed-cost-shape", "changed-effect-count", "lost-marker", "changed-keyword", "changed-group", "missing-builder", "missing-group", "empty", "invalid", "duration", "phased", "face-down", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					first := NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(1))), WithLayeredTextRevisionActivationState())
					second := NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(2))), WithLayeredTextRevisionActivationState(), WithOncePerTurn())
					if changed && scenario == "changed-timing" {
						WithSorcerySpeed()(second)
					}
					if changed && scenario == "changed-cost-shape" {
						WithCost(GenericCost(1))(second)
					}
					if changed && scenario == "lost-marker" {
						second.activationTextLayer = false
					}
					if changed && scenario == "changed-effect-count" {
						WithEffect(GainLifeFor(You, Fixed(3)))(second)
					}
					var other Ability = second
					if changed && scenario == "changed-kind" {
						other = StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)).WithLayeredPTTextRevision()
					}
					group := 1
					if changed && scenario == "changed-group" {
						group = 2
					}
					options := []CardOption{WithDefinitionAbilityGroup(0, WithAbility(first)), WithDefinitionAbilityGroup(group, WithAbility(other)), WithKeyword(core.Flying)}
					if scenario == "missing-group" {
						options[1] = WithAbility(other)
					}
					if changed && scenario == "changed-keyword" {
						options = append(options, WithKeyword(core.FirstStrike))
					}
					return NewCreature("Actions", "", 2, 3, options...)
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
			p := g.PutOnBattlefield(card, a.PlayerID())
			switch scenario {
			case "missing-second-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = p.RuntimeAbilities[:1]
			case "phased":
				g.PhaseOut(p.ID())
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
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
			before := len(g.effects.effects)
			snapshot := g.FindPermanentIncludingPhased(p.ID())
			err = g.AddPrintedActivatedObjectTextEffect(p, duration, changes...)
			if scenario == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid complete activated context accepted")
			}
			if len(g.effects.effects) != before || g.FindPermanentIncludingPhased(p.ID()) != snapshot {
				t.Fatal("failed preflight changed live state")
			}
		})
	}
}
