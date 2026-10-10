package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextMixedStaticEventObjectAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "upkeep", "missing-first-runtime", "missing-registration", "nil-runtime-event", "changed-runtime-event", "missing-second-runtime", "changed-mana-symbol", "changed-kind", "changed-timing", "changed-cost-shape", "changed-effect-count", "lost-marker", "changed-keyword", "changed-group", "missing-builder", "missing-group", "empty", "invalid", "duration", "phased", "face-down", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					first := StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, HasColorFilter(core.Red))).WithLayeredPTTextRevision()
					var second Ability = NewControllerManaBonusAbility(And(IsLand, HasSubType("Mountain")), core.Red).WithLayeredManaTapTextRevision()
					if scenario == "upkeep" || scenario == "changed-cost-shape" || scenario == "changed-timing" {
						cost := "{W}{W}"
						step := YourUpkeep
						if changed && scenario == "changed-cost-shape" {
							cost = "{U}{U}"
						}
						if changed && scenario == "changed-timing" {
							step = EachUpkeep
						}
						second = WhenEvent(TurnStepBegins(step, BindPlayer("step")), UnlessPaid(You, Costs(ManaPayment(cost)), Sacrifice(This)))
					}
					if changed && scenario == "lost-marker" {
						first.ptTextLayer = false
					}
					if changed && scenario == "changed-effect-count" {
						first.Effects = append(first.Effects, BoostAllCreaturesIncludingSelf(1, 1, IsCreature))
					}
					if changed && scenario == "changed-kind" {
						second = NewAttachedManaBonusAbility(core.Red).WithLayeredManaTapTextRevision()
					}
					if changed && scenario == "changed-mana-symbol" {
						second = NewControllerManaBonusAbility(IsLand, core.Green).WithLayeredManaTapTextRevision()
					}
					group := 1
					if changed && scenario == "changed-group" {
						group = 2
					}
					options := []CardOption{WithMixedStaticEventSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithAbility(first)), WithDefinitionAbilityGroup(group, WithAbility(second))}
					if scenario == "missing-group" {
						options[2] = WithAbility(second)
					}
					if changed && scenario == "changed-keyword" {
						options = append(options, WithKeyword(core.Flying))
					}

					return NewEnchantment("Mixed", "", options...)
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
			case "nil-runtime-event":
				g.MutablePermanent(p.ID()).RuntimeAbilities[1] = (*ManaBonusAbility)(nil)
			case "changed-runtime-event":
				event := *p.RuntimeAbilities[1].(*ManaBonusAbility)
				event.BonusMana = core.Green
				g.MutablePermanent(p.ID()).RuntimeAbilities[1] = &event
			case "missing-registration":
				id := p.RuntimeAbilities[0].AbilityID()
				for i, effect := range g.effects.effects {
					if g.printedStaticRegistration(effect, p, id) {
						g.effects.effects = append(g.effects.effects[:i], g.effects.effects[i+1:]...)
						break
					}
				}
			case "missing-first-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = p.RuntimeAbilities[1:]
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
			err = g.AddPrintedMixedStaticEventTextEffect(p, duration, changes...)
			if scenario == "valid" || scenario == "upkeep" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid mixed context accepted")
			}
			if len(g.effects.effects) != before || g.FindPermanentIncludingPhased(p.ID()) != snapshot {
				t.Fatal("failed preflight changed live state")
			}
		})
	}
}
