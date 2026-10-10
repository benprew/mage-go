package mage

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextUpkeepObjectRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "duration", "nil", "face-down", "phased", "departed", "returned", "no-builder", "no-group", "missing-runtime", "nil-runtime", "changed-runtime", "changed-player-context", "changed-capture", "changed-target", "changed-body-count", "changed-keyword", "changed-subtype", "changed-marker", "optional-rule", "grant", "lost-choice"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("event player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
					if scenario == "lost-choice" {
						trigger = WhenEvent(ChosenPlayerUpkeep(BindPlayer("event player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
					}
					if changed {
						switch scenario {
						case "changed-player-context":
							trigger = WhenEvent(TurnStepBegins(EachUpkeep, BindPlayer("event player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
						case "changed-capture":
							trigger = WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("other player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
						case "changed-target":
							trigger.AddTarget(TargetPermanent(IsCreature))
						case "changed-body-count":
							trigger.effects = append(trigger.effects, GainLifeFor(You, Fixed(1)))
						case "changed-marker":
							trigger.upkeepTextLayer = false
						}
					}
					if changed && scenario == "optional-rule" {
						trigger.Optional = true
					}
					rule := WithAbility(trigger)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(0, rule)
					}
					opts := []CardOption{WithUpkeepObjectTextRevision(), rule}
					if scenario == "lost-choice" {
						if !changed {
							opts = append(opts, WithDefinitionAbilityGroup(1, WithAbility(ChooseOpponentAsEnters())))
						}
					}

					if changed && scenario == "changed-keyword" {
						opts = append(opts, WithKeyword(core.Shroud))
					}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Shrine"))
					}
					return NewEnchantment("Upkeep event", "", opts...)
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
			case "grant":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &grantedByEffect{Ability: p.RuntimeAbilities[0]}
			case "changed-runtime":
				fresh := *p.RuntimeAbilities[0].(*GenericTriggered)
				fresh.targets = []Target{TargetPermanent(IsCreature)}
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &fresh
			}
			before := len(g.effects.effects)
			rulesBefore := append([]Ability(nil), upkeepObjectRuntimeRules(p)...)
			if err := g.AddPrintedUpkeepObjectTextEffect(p, duration, changes...); err == nil {
				t.Fatal("invalid complete context accepted")
			}
			if !slices.Equal(upkeepObjectRuntimeRules(p), rulesBefore) {
				t.Fatal("failed preflight changed declarations")
			}
			if len(g.effects.effects) != before {
				t.Fatal("failed preflight installed an effect")
			}
		})
	}
}

func TestTextUpkeepObjectDeclarations(t *testing.T) {
	for _, scenario := range []string{"valid", "choice", "missing-marker", "duplicate-marker", "missing-trigger", "keyword", "sibling", "duplicate-choice", "unbound-choice", "unused-choice", "land", "aura", "casting-cost", "casting-target", "nil-rule", "end-step", "optional"} {
		t.Run(scenario, func(t *testing.T) {
			var pattern EventPattern = TurnStepBegins(YourUpkeep, BindPlayer("player"))
			if scenario == "choice" || scenario == "unbound-choice" || scenario == "duplicate-choice" {
				pattern = ChosenPlayerUpkeep(BindPlayer("player"))
			}
			trigger := WhenEvent(pattern, GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
			if scenario == "end-step" {
				trigger.eventType = core.EvtEndStep
			}
			if scenario == "optional" {
				trigger.Optional = true
			}
			opts := make([]CardOption, 0, 8)
			opts = append(opts, WithAbility(trigger))
			if scenario != "missing-marker" {
				opts = append(opts, WithUpkeepObjectTextRevision())
			}
			switch scenario {
			case "choice", "unused-choice":
				opts = append(opts, WithAbility(ChooseOpponentAsEnters()))
			case "duplicate-choice":
				opts = append(opts, WithAbility(ChooseOpponentAsEnters()), WithAbility(ChooseOpponentAsEnters()))
			case "duplicate-marker":
				opts = append(opts, WithUpkeepObjectTextRevision())
			case "missing-trigger":
				opts = opts[:0]
				opts = append(opts, WithUpkeepObjectTextRevision())
			case "keyword":
				opts = append(opts, WithKeyword(core.Trample))
			case "sibling":
				opts = append(opts, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
			case "land":
				opts = append(opts, WithCardType(core.TypeLand))
			case "aura":
				opts = append(opts, WithSubTypes("Aura"))
			case "casting-cost":
				opts = append(opts, WithAdditionalCost(&ManaCostPayment{MC: core.ManaCost{Generic: 1}}))
			case "casting-target":
				opts = append(opts, WithCastTarget(TargetCreature()))
			case "nil-rule":
				opts = append(opts, WithAbility((*GenericTriggered)(nil)))
			}
			card := NewArtifact("Declaration", "", opts...)
			_, err := upkeepObjectTextContext(card)
			valid := scenario == "valid" || scenario == "choice"
			if (err == nil) != valid {
				t.Fatalf("context: %v", err)
			}
			if scenario != "missing-marker" && (ValidateCardDefinition(card.Name(), card) == nil) != valid {
				t.Fatal("definition validation did not check the complete context")
			}
		})
	}
}

func upkeepObjectRuntimeRules(p *Permanent) []Ability {
	if p == nil {
		return nil
	}
	return p.RuntimeAbilities
}
