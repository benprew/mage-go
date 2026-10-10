package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextProtectionPTSpellDeclaration(t *testing.T) {
	valid := func(effect ContinuousEffect, extra ...CardOption) Card {
		opts := make([]CardOption, 0, 3+len(extra))
		opts = append(opts, WithProtectionPTSpellTextContinuation(), WithAbility(ProtectionFromColors(core.Red).WithLayeredColorTextRevision()), WithAbility(StaticAbility(effect).WithLayeredPTTextRevision()))
		return NewCreature("Mixed", "", 3, 3, append(opts, extra...)...)
	}
	for _, effect := range []ContinuousEffect{SourcePTWhileOpponentControlsNontokenColor(core.White, 1, 1), NamedCreaturesPTWhileOpponentControlsNontokenColor("Mixed", core.Red, 1, 1)} {
		if err := ValidateCardDefinition("Mixed", valid(effect)); err != nil {
			t.Fatal(err)
		}
	}
	for _, card := range []Card{
		valid(SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithProtectionPTSpellTextContinuation()),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithIntrinsicCreatureSpellTextContinuation()),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithKeyword(core.Flying)),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithAdditionalCost(LifePayCost(1))),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithCastTarget(TargetPermanent(IsCreature))),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithSubTypes("Aura")),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithCardType(core.TypeLand)),
		valid(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), WithAbility((*StaticAbilityHolder)(nil))),
		NewCreature("Unmarked", "", 3, 3, WithProtectionPTSpellTextContinuation(), WithAbility(ProtectionFromColors(core.Red)), WithAbility(StaticAbility(SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1)).WithLayeredPTTextRevision())),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestTextProtectionPTSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-kind", "changed-recipient", "changed-protection", "changed-keyword", "changed-subtype", "changed-name", "unmarked", "swapped", "saved-result", "targets", "body", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					effect := SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1)
					if scenario == "changed-name" {
						effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Static", core.Red, 1, 1)
					}
					if changed && scenario == "changed-kind" {
						effect = SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)
					}
					if changed && scenario == "changed-recipient" {
						effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Static", core.Red, 1, 1)
					}
					if changed && scenario == "changed-name" {
						effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Other", core.Red, 1, 1)
					}
					protection := ProtectionFromColors(core.Red).WithLayeredColorTextRevision()
					if changed && scenario == "changed-protection" {
						protection = ProtectionFromColors(core.Red, core.Blue).WithLayeredColorTextRevision()
					}
					static := StaticAbility(effect).WithLayeredPTTextRevision()
					if changed && scenario == "unmarked" {
						static = StaticAbility(effect)
					}
					rule := WithAbility(static)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(1, rule)
					}
					opts := []CardOption{WithProtectionPTSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithAbility(protection)), rule}
					if changed && scenario == "swapped" {
						opts = []CardOption{WithProtectionPTSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithAbility(static)), WithDefinitionAbilityGroup(1, WithAbility(protection))}
					}
					if changed && scenario == "changed-keyword" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Shrine"))
					}
					return NewCreature("Static", "", 5, 5, opts...)
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
