package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextUpkeepSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 2+len(extra))
		opts = append(opts, WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("event player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()))
		return NewEnchantment("Static", "", append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Static", valid()); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		valid(WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision()), valid(WithAdditionalCost(LifePayCost(1))),
		valid(WithManaTapSpellTextContinuation()), valid(WithMixedStaticEventSpellTextContinuation()),
		valid(WithKeyword(core.Islandwalk)), valid(WithSubTypes("Aura")),
		valid(WithGlobalStaticSpellTextContinuation()), valid(WithIntrinsicCreatureSpellTextContinuation()),
		valid(WithCastTarget(TargetPermanent(IsCreature))), valid(WithAbility((*StaticAbilityHolder)(nil))),
		valid(WithAbility(NewActivated(TapCost(This), GainLifeFor(You, Fixed(1))))),
		NewInstant("Wrong type", "", WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision()),
		NewEnchantment("No rule", "", WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision()),
		NewEnchantment("Nil rule", "", WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), WithAbility((*GenericTriggered)(nil))),
		NewEnchantment("Unmarked trigger", "", WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("event player")), GainLifeFor(You, Fixed(1))))),
		NewEnchantment("Unmarked", "", WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), WithAbility(StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)))),
		NewAura("Host", "", WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp")).WithLayeredLandTypeTextRevision())),
		NewEnchantment("Conditional", "", WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), WithAbility(StaticAbility(SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)).WithLayeredPTTextRevision())),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid marker on %s", card.Name())
		}
	}
}

func TestTextUpkeepSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-kind", "changed-capture", "changed-target", "changed-keyword", "changed-subtype", "saved-result", "targets", "body", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					holder := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("event player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
					if changed && scenario == "changed-kind" {
						holder = WhenEvent(TurnStepBegins(EachUpkeep, BindPlayer("event player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
					}
					if changed && scenario == "changed-capture" {
						holder = WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("other player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
					}
					if changed && scenario == "changed-target" {
						holder.AddTarget(TargetPermanent(IsCreature))
					}
					rule := WithAbility(holder)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(0, rule)
					}
					opts := []CardOption{WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision(), rule}
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
			g.SetStep(core.PrecombatMain)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			switch scenario {
			case "saved-result":
				before.capturedNumbers = map[*NumberBinding]int{BindNumber("foreign"): 1}
			case "targets":
				before.TargetSpecs = []Target{TargetPermanent(IsCreature)}
			case "body":
				before.Effects = []Effect{GainLifeFor(You, Fixed(1))}
			}
			change, _ := ReplaceColorWord(core.Black, core.Blue)
			changes := []TextChange{change}
			if scenario == "invalid" {
				changes = []TextChange{{}}
			}
			if scenario == "empty" {
				changes = nil
			}
			if scenario == "countered" {
				g.CounterSpellOnStack(card.ID())
			}
			if err := g.RevisePrintedSpellText(before.ID, changes...); err == nil {
				t.Fatal("invalid continuation accepted")
			}
			if scenario == "countered" {
				if len(a.Graveyard()) != 1 || g.StackSize() != 0 {
					t.Fatal("revision changed departure")
				}
				return
			}
			if g.StackPeek() != before {
				t.Fatal("rejection mutated stack visit")
			}
		})
	}
}
