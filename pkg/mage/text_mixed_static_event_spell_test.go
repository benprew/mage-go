package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextMixedStaticEventSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 2+len(extra))
		opts = append(opts, WithMixedStaticEventSpellTextContinuation(), WithAbility(StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, HasColorFilter(core.Black))).WithLayeredPTTextRevision()), WithAbility(NewControllerManaBonusAbility(IsLand, core.Red).WithLayeredManaTapTextRevision()))
		return NewEnchantment("Static", "", append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Static", valid()); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		valid(WithMixedStaticEventSpellTextContinuation()), valid(WithAdditionalCost(LifePayCost(1))),
		valid(WithKeyword(core.Shroud)), valid(WithSubTypes("Aura")),
		valid(WithGlobalStaticSpellTextContinuation()), valid(WithIntrinsicCreatureSpellTextContinuation()),
		valid(WithCastTarget(TargetPermanent(IsCreature))), valid(WithAbility((*StaticAbilityHolder)(nil))),
		valid(WithAbility(NewActivated(TapCost(This), GainLifeFor(You, Fixed(1))))),
		NewInstant("Wrong type", "", WithMixedStaticEventSpellTextContinuation()),
		NewEnchantment("No rule", "", WithMixedStaticEventSpellTextContinuation()),
		NewEnchantment("Nil rule", "", WithMixedStaticEventSpellTextContinuation(), WithAbility((*GenericTriggered)(nil))),
		NewEnchantment("Unmarked trigger", "", WithMixedStaticEventSpellTextContinuation(), WithAbility(WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))))),
		NewEnchantment("Unmarked", "", WithMixedStaticEventSpellTextContinuation(), WithAbility(StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)))),
		NewAura("Host", "", WithMixedStaticEventSpellTextContinuation(), WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp")).WithLayeredLandTypeTextRevision())),
		NewEnchantment("Conditional", "", WithMixedStaticEventSpellTextContinuation(), WithAbility(StaticAbility(SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)).WithLayeredPTTextRevision())),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid marker on %s", card.Name())
		}
	}
}

func TestTextMixedStaticEventSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-kind", "changed-capture", "changed-target", "changed-keyword", "changed-subtype", "saved-result", "targets", "body", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					holder := NewControllerManaBonusAbility(IsLand, core.Red).WithLayeredManaTapTextRevision()
					if changed && scenario == "changed-kind" {
						holder = NewAttachedManaBonusAbility(core.Red).WithLayeredManaTapTextRevision()
					}
					if changed && scenario == "changed-capture" {
						holder = NewManaBonusAbility(IsLand, core.Red).WithLayeredManaTapTextRevision()
					}
					if changed && scenario == "changed-target" {
						holder.MatchProduced = true
					}
					rule := WithAbility(holder)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(0, rule)
					}
					opts := []CardOption{WithMixedStaticEventSpellTextContinuation(), WithDefinitionAbilityGroup(1, WithAbility(StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)).WithLayeredPTTextRevision())), rule}
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

func TestTextMixedStaticEventSpellUpkeepDeclaration(t *testing.T) {
	valid := func(rule Ability, extra ...CardOption) Card {
		opts := make([]CardOption, 0, 3+len(extra))
		opts = append(opts, WithMixedStaticEventSpellTextContinuation(), WithAbility(StaticAbility(BecomesBasicLandsEffect(And(IsLand, HasSubType("Mountain")), "Plains")).WithLayeredLandTypeTextRevision()), WithAbility(rule))
		return NewEnchantment("Mixed", "", append(opts, extra...)...)
	}
	makeRule := func(effect Effect) *GenericTriggered {
		return WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("step")), effect).(*GenericTriggered)
	}
	upkeep := func() *GenericTriggered {
		return makeRule(UnlessPaid(You, Costs(ManaPayment("{W}{W}")), Sacrifice(This)))
	}
	if err := ValidateCardDefinition("Mixed", valid(upkeep())); err != nil {
		t.Fatal(err)
	}
	for _, rule := range []Ability{
		(*GenericTriggered)(nil),
		makeRule(UnlessPaid(FromPlayer(BindPlayer("other")), Costs(ManaPayment("{W}{W}")), Sacrifice(This))),
		makeRule(IfPaid(You, Costs(ManaPayment("{W}{W}")), Sacrifice(This))),
		makeRule(UnlessPaid(You, Costs(ManaPayment("{X}")), Sacrifice(This))),
		makeRule(UnlessPaid(You, Costs(LifePayCost(1)), Sacrifice(This))),
		makeRule(UnlessPaid(You, Costs(ManaPayment("{W}{W}")), Sacrifice(AttachedPermanent))),
		makeRule(UnlessPaid(You, Costs(ManaPayment("{W}{W}")), Sacrifice(This, SacrificeAmountInto(BindNumber("foreign"))))),
		upkeep().FromGraveyard(), upkeep().InZone(core.ZoneHand),
		WhenEvent(TurnStepBegins(EachUpkeep, BindPlayer("step")), UnlessPaid(You, Costs(ManaPayment("{W}{W}")), Sacrifice(This))),
	} {
		if err := ValidateCardDefinition("Mixed", valid(rule)); err == nil {
			t.Fatal("invalid upkeep pair accepted")
		}
	}
}

func TestTextMixedStaticEventSpellUpkeepRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"mana-symbol", "event-capture", "static-kind", "group-order"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					cost := "{W}{W}"
					player := BindPlayer("step")
					static := StaticAbility(BecomesBasicLandsEffect(And(IsLand, HasSubType("Mountain")), "Plains")).WithLayeredLandTypeTextRevision()
					if changed {
						switch scenario {
						case "mana-symbol":
							cost = "{U}{U}"
						case "event-capture":
							player = nil
						case "static-kind":
							static = StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)).WithLayeredPTTextRevision()
						}
					}
					rule := WhenEvent(TurnStepBegins(YourUpkeep, player), UnlessPaid(You, Costs(ManaPayment(cost)), Sacrifice(This)))
					first, second := WithDefinitionAbilityGroup(0, WithAbility(static)), WithDefinitionAbilityGroup(1, WithAbility(rule))
					if changed && scenario == "group-order" {
						first, second = second, first
					}
					return NewEnchantment("Mixed", "", WithMixedStaticEventSpellTextContinuation(), first, second)
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
			change, _ := ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
			if err := g.RevisePrintedSpellText(before.ID, change); err == nil || g.StackPeek() != before {
				t.Fatal("changed pair context mutated stack")
			}
		})
	}
}
