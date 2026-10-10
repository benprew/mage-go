package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextSpellCastObjectRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "duration", "nil", "face-down", "phased", "departed", "returned", "no-builder", "no-group", "missing-runtime", "nil-runtime", "changed-runtime", "changed-caster", "changed-capture", "changed-target", "changed-body-count", "changed-keyword", "changed-subtype", "changed-marker", "optional-rule"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					trigger := WhenEvent(SpellCast().OfColor(core.Black), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision()
					if changed {
						switch scenario {
						case "changed-caster":
							trigger = WhenEvent(SpellCast().ByOpponent(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision()
						case "changed-capture":
							trigger = WhenEvent(SpellCast().WithCaster(BindPlayer("caster")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision()
						case "changed-target":
							trigger.AddTarget(TargetPermanent(IsCreature))
						case "changed-body-count":
							trigger.effects = append(trigger.effects, GainLifeFor(You, Fixed(1)))
						case "changed-marker":
							trigger.spellCastTextLayer = false
						}
					}
					if scenario == "optional-rule" {
						trigger.Optional = true
					}
					rule := WithAbility(trigger)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(0, rule)
					}
					opts := []CardOption{WithSpellCastSpellTextContinuation(), rule}
					if changed && scenario == "changed-keyword" {
						opts = append(opts, WithKeyword(core.Shroud))
					}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Shrine"))
					}
					return NewEnchantment("Cast event", "", opts...)
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
			changes := []TextChange{change}
			duration := core.Indefinite
			switch scenario {
			case "empty":
				changes = nil
			case "invalid":
				changes = []TextChange{{}}
			case "duration":
				duration = core.EndOfCombat
			case "nil":
				p = nil
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "departed":
				g.RemoveFromBattlefield(p)
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "missing-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = nil
			case "nil-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = (*GenericTriggered)(nil)
			case "changed-runtime":
				fresh := *p.RuntimeAbilities[0].(*GenericTriggered)
				fresh.targets = []Target{TargetPermanent(IsCreature)}
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &fresh
			}
			before := len(g.effects.effects)
			if err := g.AddPrintedSpellCastObjectTextEffect(p, duration, changes...); err == nil {
				t.Fatal("invalid complete context accepted")
			}
			if len(g.effects.effects) != before {
				t.Fatal("failed preflight installed an effect")
			}
		})
	}
}
