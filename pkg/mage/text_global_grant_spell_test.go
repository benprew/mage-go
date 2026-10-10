package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextGlobalGrantSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 2+len(extra))
		opts = append(opts, WithGlobalStaticSpellTextContinuation(), WithAbility(StaticAbility(GrantKeywordToAllIncludingSource(core.Forestwalk, IsCreature), BoostAllCreaturesIncludingSelf(1, 1, IsCreature)).WithLayeredGrantTextRevision()))
		return NewEnchantment("Grant", "", append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Grant", valid()); err != nil {
		t.Fatal(err)
	}
	for _, extra := range []CardOption{WithGlobalStaticSpellTextContinuation(), WithIntrinsicCreatureSpellTextContinuation(), WithKeyword(core.Flying), WithAdditionalCost(LifePayCost(1)), WithCastTarget(TargetPermanent(IsCreature)), WithSubTypes("Aura"), WithAbility((*StaticAbilityHolder)(nil))} {
		if err := ValidateCardDefinition("Grant", valid(extra)); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	for _, holder := range []*StaticAbilityHolder{
		StaticAbility(GrantKeywordToAllIncludingSource(core.Forestwalk, IsCreature)),
		StaticAbility(GrantKeywordToAllIncludingSource(core.Forestwalk, IsCreature)).WithGrantTextRevision(),
		StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)).WithLayeredGrantTextRevision(),
	} {
		card := NewEnchantment("Invalid", "", WithGlobalStaticSpellTextContinuation(), WithAbility(holder))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid grant marker accepted")
		}
	}
}

func TestTextGlobalGrantSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"changed-count", "changed-order", "changed-kind", "changed-marker", "changed-subtype", "changed-group", "no-builder", "no-group", "invalid", "saved-result", "targets", "body", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					effects := []ContinuousEffect{GrantKeywordToAllIncludingSource(core.Forestwalk, IsCreature), BoostAllCreaturesIncludingSelf(1, 1, IsCreature)}
					if changed && scenario == "changed-count" {
						effects = effects[:1]
					}
					if changed && scenario == "changed-order" {
						effects[0], effects[1] = effects[1], effects[0]
					}
					holder := StaticAbility(effects...).WithLayeredGrantTextRevision()
					if changed && scenario == "changed-kind" {
						holder = StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)).WithLayeredPTTextRevision()
					}
					if changed && scenario == "changed-marker" {
						holder.grantTextLayer = false
					}
					group := 0
					if changed && scenario == "changed-group" {
						group = 1
					}
					rule := WithDefinitionAbilityGroup(group, WithAbility(holder))
					if scenario == "no-group" {
						rule = WithAbility(holder)
					}
					opts := []CardOption{WithGlobalStaticSpellTextContinuation(), rule}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Shrine"))
					}
					return NewEnchantment("Grant", "", opts...)
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
			case "countered":
				g.CounterSpellOnStack(card.ID())
			}
			change, _ := ReplaceBasicLandWord(core.ForestLandType, core.SwampLandType)
			if scenario == "invalid" {
				change = TextChange{}
			}
			if err := g.RevisePrintedSpellText(before.ID, change); err == nil {
				t.Fatal("invalid continuation accepted")
			}
			if scenario == "countered" {
				if g.StackSize() != 0 || len(a.Graveyard()) != 1 {
					t.Fatal("revision changed departure")
				}
			} else if g.StackPeek() != before {
				t.Fatal("rejection changed stack visit")
			}
		})
	}
}
