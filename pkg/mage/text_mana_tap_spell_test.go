package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextManaTapSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 2+len(extra))
		opts = append(opts, WithManaTapSpellTextContinuation(), WithAbility(WhenEvent(LandTappedForMana().OfManaType(core.Black), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()))
		return NewEnchantment("Static", "", append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Static", valid()); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		valid(WithManaTapSpellTextContinuation()), valid(WithAdditionalCost(LifePayCost(1))),
		valid(WithKeyword(core.Shroud)), valid(WithSubTypes("Aura")),
		valid(WithGlobalStaticSpellTextContinuation()), valid(WithIntrinsicCreatureSpellTextContinuation()),
		valid(WithCastTarget(TargetPermanent(IsCreature))), valid(WithAbility((*StaticAbilityHolder)(nil))),
		valid(WithAbility(NewActivated(TapCost(This), GainLifeFor(You, Fixed(1))))),
		NewInstant("Wrong type", "", WithManaTapSpellTextContinuation()),
		NewEnchantment("No rule", "", WithManaTapSpellTextContinuation()),
		NewEnchantment("Nil rule", "", WithManaTapSpellTextContinuation(), WithAbility((*GenericTriggered)(nil))),
		NewEnchantment("Unmarked trigger", "", WithManaTapSpellTextContinuation(), WithAbility(WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))))),
		NewEnchantment("Unmarked", "", WithManaTapSpellTextContinuation(), WithAbility(StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)))),
		NewAura("Host", "", WithManaTapSpellTextContinuation(), WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp")).WithLayeredLandTypeTextRevision())),
		NewEnchantment("Conditional", "", WithManaTapSpellTextContinuation(), WithAbility(StaticAbility(SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)).WithLayeredPTTextRevision())),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid marker on %s", card.Name())
		}
	}
}

func TestTextManaTapSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-kind", "changed-capture", "changed-target", "changed-keyword", "changed-subtype", "saved-result", "targets", "body", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					holder := WhenEvent(LandTappedForMana().OfManaType(core.Black), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
					if changed && scenario == "changed-kind" {
						holder = WhenEvent(LandTappedForMana().OfAttachedLand(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
					}
					if changed && scenario == "changed-capture" {
						holder = WhenEvent(LandTappedForMana().WithPlayer(BindPlayer("producer")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
					}
					if changed && scenario == "changed-target" {
						holder.AddTarget(TargetPermanent(IsCreature))
					}
					rule := WithAbility(holder)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(0, rule)
					}
					opts := []CardOption{WithManaTapSpellTextContinuation(), rule}
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

func TestTextManaTapSpellBonusDeclaration(t *testing.T) {
	for _, rule := range []*ManaBonusAbility{
		NewManaFlareAbility(IsLand).WithLayeredManaTapTextRevision(),
		NewManaBonusAbility(HasSubType("Mountain"), core.Red).WithLayeredManaTapTextRevision(),
		NewControllerManaBonusAbility(HasSubType("Mountain"), core.Red).WithLayeredManaTapTextRevision(),
	} {
		card := NewEnchantment("Bonus", "", WithManaTapSpellTextContinuation(), WithAbility(rule))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, rule := range []Ability{
		(*ManaBonusAbility)(nil), NewManaFlareAbility(IsLand),
		NewAttachedManaBonusAbility(core.Green).WithLayeredManaTapTextRevision(),
		WhenEvent(LandTappedForMana().OfAttachedLand(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision(),
		WhenEvent(SpellCast(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision(),
	} {
		card := NewEnchantment("Invalid", "", WithManaTapSpellTextContinuation(), WithAbility(rule))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid mana rule accepted")
		}
	}
}

func TestTextManaTapSpellBonusRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"produced-context", "recipient-context", "ability-kind"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					var rule Ability = NewManaFlareAbility(IsLand).WithLayeredManaTapTextRevision()
					if changed {
						switch scenario {
						case "produced-context":
							rule = NewManaBonusAbility(IsLand, core.Blue).WithLayeredManaTapTextRevision()
						case "recipient-context":
							bonus := NewManaFlareAbility(IsLand).WithLayeredManaTapTextRevision()
							bonus.controllerRecipient = true
							rule = bonus
						case "ability-kind":
							rule = WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
						}
					}
					return NewEnchantment("Bonus", "", WithManaTapSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithAbility(rule)))
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
			WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			g, a, _ := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			change, _ := ReplaceColorWord(core.Red, core.Blue)
			if err := g.RevisePrintedSpellText(before.ID, change); err == nil || g.StackPeek() != before {
				t.Fatal("changed bonus context mutated stack")
			}
		})
	}
}
