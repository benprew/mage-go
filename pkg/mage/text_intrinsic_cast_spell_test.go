package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextIntrinsicCastSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 3+len(extra))
		opts = append(opts, WithIntrinsicCastEventSpellTextContinuation(), WithLayeredPrintedBasicLandwalk(core.Swampwalk), WithAbility(WhenEvent(SpellCast().OfColor(core.Black), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision()))
		return NewCreature("Mixed", "", 5, 5, append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Mixed", valid()); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		valid(WithIntrinsicCastEventSpellTextContinuation()), valid(WithIntrinsicCreatureSpellTextContinuation()), valid(WithSpellCastSpellTextContinuation()),
		valid(WithAdditionalCost(LifePayCost(1))), valid(WithCastTarget(TargetPermanent(IsCreature))), valid(WithSubTypes("Aura")), valid(WithCardType(core.TypeLand)),
		valid(WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, Fixed(1))))), valid(WithAbility((*GenericTriggered)(nil))),
		NewCreature("No intrinsic", "", 5, 5, WithIntrinsicCastEventSpellTextContinuation(), WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision())),
		NewCreature("Unmarked intrinsic", "", 5, 5, WithIntrinsicCastEventSpellTextContinuation(), WithKeyword(core.Swampwalk), WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision())),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid marker on %s", card.Name())
		}
	}
}

func TestTextIntrinsicCastSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-kind", "changed-capture", "changed-target", "changed-keyword", "changed-subtype", "saved-result", "targets", "body", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					holder := WhenEvent(SpellCast().OfColor(core.Black), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision()
					if changed && scenario == "changed-kind" {
						holder = WhenEvent(SpellCast().ByOpponent(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision()
					}
					if changed && scenario == "changed-capture" {
						holder = WhenEvent(SpellCast().WithCaster(BindPlayer("caster")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredSpellCastTextRevision()
					}
					if changed && scenario == "changed-target" {
						holder.AddTarget(TargetPermanent(IsCreature))
					}
					rule := WithAbility(holder)
					if scenario != "no-group" {
						rule = WithDefinitionAbilityGroup(1, rule)
					}
					opts := []CardOption{WithIntrinsicCastEventSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithLayeredPrintedBasicLandwalk(core.Swampwalk)), rule}
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
