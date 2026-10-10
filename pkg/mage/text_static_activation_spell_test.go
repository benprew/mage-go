package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextStaticActivationSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 3+len(extra))
		opts = append(opts, WithStaticActivationSpellTextContinuation(), WithAbility(StaticAbility(SourcePTWhileControllerControlsBasicLand("Swamp", 1, 1)).WithLayeredPTTextRevision()), WithAbility(NewAction(ActionActivated, WithCost(ManaCostOf("B")), WithLayeredTextRevisionActivationState(), WithEffect(Regenerate(This)))))
		return NewCreature("Mixed", "2R", 2, 2, append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Mixed", valid()); err != nil {
		t.Fatal(err)
	}
	opponent := NewCreature("Mixed", "", 2, 2, WithStaticActivationSpellTextContinuation(), WithAbility(NewAction(ActionActivated, WithLayeredTextRevisionActivationState(), WithEffect(Regenerate(This)))), WithAbility(StaticAbility(SourcePTWhileOpponentControlsBasicLand("Island", 1, 1)).WithLayeredPTTextRevision()))
	if err := ValidateCardDefinition("Mixed", opponent); err != nil {
		t.Fatal(err)
	}
	for _, extra := range []CardOption{WithStaticActivationSpellTextContinuation(), WithGlobalStaticSpellTextContinuation(), WithProtectionPTSpellTextContinuation(), WithKeyword(core.Flying), WithAdditionalCost(LifePayCost(1)), WithCastTarget(TargetPermanent(IsCreature)), WithSubTypes("Aura"), WithCardType(core.TypeLand), WithAbility((*StaticAbilityHolder)(nil))} {
		if err := ValidateCardDefinition("Mixed", valid(extra)); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	for _, action := range []*ActionDefinition{
		nil,
		NewAction(ActionSpell, WithLayeredTextRevisionActivationState()),
		NewAction(ActionActivated, WithEffect(Regenerate(This))),
		NewAction(ActionActivated, WithLayeredTextRevisionActivationState(), WithActivationHistory(BindActivationHistory("history"))),
	} {
		card := NewCreature("Invalid", "", 2, 2, WithStaticActivationSpellTextContinuation(), WithAbility(StaticAbility(SourcePTWhileControllerControlsBasicLand("Swamp", 1, 1)).WithLayeredPTTextRevision()), WithAbility(action))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid activated sibling accepted")
		}
	}
}

func TestTextStaticActivationSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-condition", "changed-recipient", "changed-subtype", "changed-keyword", "changed-cost-count", "changed-timing", "changed-target", "changed-body-count", "unmarked-static", "unmarked-action", "swapped", "saved-result", "targets", "body", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					effect := SourcePTWhileControllerControlsBasicLand("Swamp", 1, 1)
					if changed && scenario == "changed-condition" {
						effect = SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)
					}
					if changed && scenario == "changed-recipient" {
						effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Mixed", core.Red, 1, 1)
					}
					static := StaticAbility(effect).WithLayeredPTTextRevision()
					if changed && scenario == "unmarked-static" {
						static = StaticAbility(effect)
					}
					actionOptions := []ActionOption{WithCost(ManaCostOf("B")), WithEffect(Regenerate(This)), WithLayeredTextRevisionActivationState()}
					if changed && scenario == "unmarked-action" {
						actionOptions = actionOptions[:2]
					}
					if changed && scenario == "changed-cost-count" {
						actionOptions = append(actionOptions, WithCost(GenericCost(1)))
					}
					if changed && scenario == "changed-timing" {
						actionOptions = append(actionOptions, WithSorcerySpeed())
					}
					if changed && scenario == "changed-target" {
						actionOptions = append(actionOptions, WithTarget(TargetPermanent(IsCreature)))
					}
					if changed && scenario == "changed-body-count" {
						actionOptions = append(actionOptions, WithEffect(GainLifeFor(You, Fixed(1))))
					}
					action := NewAction(ActionActivated, actionOptions...)
					staticOption, actionOption := WithDefinitionAbilityGroup(1, WithAbility(static)), WithDefinitionAbilityGroup(2, WithAbility(action))
					if scenario == "no-group" {
						actionOption = WithAbility(action)
					}
					if changed && scenario == "swapped" {
						staticOption, actionOption = WithDefinitionAbilityGroup(2, WithAbility(static)), WithDefinitionAbilityGroup(1, WithAbility(action))
					}
					opts := []CardOption{WithStaticActivationSpellTextContinuation(), staticOption, actionOption}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Troll"))
					}
					if changed && scenario == "changed-keyword" {
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
			change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			changes := []TextChange{change}
			if scenario == "empty" {
				changes = nil
			}
			if scenario == "invalid" {
				changes = []TextChange{{}}
			}
			if scenario == "countered" {
				g.CounterSpellOnStack(card.ID())
			}
			if err := g.RevisePrintedSpellText(before.ID, changes...); err == nil {
				t.Fatal("invalid continuation accepted")
			}
			if scenario == "countered" {
				if g.StackSize() != 0 || len(a.Graveyard()) != 1 {
					t.Fatal("revision changed departure")
				}
			} else if g.StackPeek() != before {
				t.Fatal("rejection changed the stack visit")
			}
		})
	}
}
