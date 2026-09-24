package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestGrantedActivationComposition(t *testing.T) {
	saved := dsl.BindPermanent("recipient")
	card := dsl.NewEnchantment("Grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.WithoutPermanent(dsl.ControlledPermanents(dsl.IsCreature), dsl.This),
		dsl.Costs(dsl.Mana("{B}")),
		dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.Regenerate(dsl.From(saved))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedTargetActivationComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetYouControl())
	var option dsl.GrantedActivationOption = dsl.WithTargets(target)
	card := dsl.NewEnchantment("Target grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(dsl.Mana("{B}")),
		dsl.Regenerate(target), option,
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedTimingComposition(t *testing.T) {
	for _, option := range []dsl.ActivationTimingOption{dsl.OnlyAsSorcery(), dsl.OnlyDuring(core.Upkeep), dsl.OnlyOnYourTurn()} {
		target := dsl.TargetPermanent("creature", dsl.IsCreature)
		card := dsl.NewEnchantment("Timed grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
			dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), dsl.Regenerate(target), dsl.WithTargets(target), option,
		))), dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This), option)))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrantedConditionComposition(t *testing.T) {
	var option = dsl.ActivationIf(dsl.SourceHasCounters(core.Charge, 1))
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewEnchantment("Conditional grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), dsl.Regenerate(target),
		dsl.WithTargets(target), dsl.OnlyAsSorcery(), option,
	))), dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This), option)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedLimitComposition(t *testing.T) {
	var option = dsl.ActivationLimit(2)
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewEnchantment("Limited grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), dsl.Regenerate(target),
		dsl.WithTargets(target), dsl.OnlyAsSorcery(), dsl.ActivationIf(dsl.Controls(dsl.IsCreature)), option,
	))), dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This), option)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedModesComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("source")
	card := dsl.NewEnchantment("Modal grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedModes(
		dsl.WithoutPermanent(dsl.MatchingPermanents(dsl.IsCreature), dsl.This),
		dsl.Costs(dsl.Mana("{1}"), dsl.TapCost(dsl.This)),
		dsl.ModeOf("regenerate", dsl.Regenerate(target), dsl.WithTargets(target)),
		dsl.ModeOf("counter", dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.AddCounters(dsl.From(saved), core.Charge, dsl.Fixed(1)))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestModalRestrictionsComposition(t *testing.T) {
	options := []dsl.ModalActivationOption{
		dsl.ModeOf("one", dsl.GainLife(dsl.You, dsl.Fixed(1))),
		dsl.OnlyAsSorcery(), dsl.ActivationIf(dsl.SourceHasCounters(core.Charge, 1)), dsl.ActivationLimit(1),
		dsl.ModeOf("two", dsl.GainLife(dsl.You, dsl.Fixed(2))),
	}
	card := dsl.NewEnchantment("Modal restrictions", "",
		dsl.WithAbility(dsl.ActivateModes(dsl.Costs(), options...)),
		dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedModes(dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), options...))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedSubjectComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetYouControl())
	card := dsl.NewAura("Subject grant", "", dsl.WithAbility(dsl.Static(
		dsl.GrantActivatedAbility(dsl.AttachedPermanent, dsl.Costs(dsl.Mana("{B}")), dsl.Regenerate(target), dsl.WithTargets(target), dsl.OnlyAsSorcery(), dsl.ActivationLimit(1)),
		dsl.GrantActivatedModes(dsl.This, dsl.Costs(), dsl.ActivationIf(dsl.Controls(dsl.IsCreature)),
			dsl.ModeOf("one", dsl.GainLife(dsl.You, dsl.Fixed(1))),
			dsl.ModeOf("two", dsl.GainLife(dsl.You, dsl.Fixed(2)))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedModificationComposition(t *testing.T) {
	target := dsl.TargetPermanent("recipient", dsl.IsCreature)
	inner := dsl.TargetPermanent("regenerate", dsl.IsCreature, dsl.TargetYouControl())
	card := dsl.NewInstant("Grant", "", dsl.WithAbility(dsl.Spell(
		dsl.Until(core.Indefinite, dsl.ActivatedAbilityModification(target, dsl.Costs(dsl.RemoveCountersCost(core.Matrix, 1)), dsl.Regenerate(inner), dsl.WithTargets(inner), dsl.OnlyAsSorcery(), dsl.ActivationLimit(1))), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	invalid := dsl.NewInstant("Invalid grant", "", dsl.WithAbility(dsl.Spell(
		dsl.Until(core.Indefinite, dsl.ActivatedAbilityModification(target, dsl.Costs(), dsl.Regenerate(target))), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("grant body read outer target")
	}
}

func TestGrantedModalModificationComposition(t *testing.T) {
	outer := dsl.TargetPermanent("recipient", dsl.IsCreature)
	inner := dsl.TargetPermanent("return", dsl.IsCreature)
	modification := dsl.ActivatedModesModification(outer, dsl.Costs(),
		dsl.ModeOf("return", dsl.ReturnToHand(inner), dsl.WithTargets(inner)),
		dsl.ModeOf("gain", dsl.GainLife(dsl.You, dsl.Fixed(2))),
		dsl.OnlyAsSorcery(), dsl.ActivationLimit(1))
	card := dsl.NewInstant("Modal grant", "", dsl.WithAbility(dsl.Spell(dsl.Until(core.EndOfTurn, modification), dsl.WithTargets(outer))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []dsl.ContinuousModification{
		dsl.ActivatedModesModification(outer, dsl.Costs(),
			dsl.ModeOf("outer read", dsl.ReturnToHand(outer)),
			dsl.ModeOf("gain", dsl.GainLife(dsl.You, dsl.Fixed(2)))),
		dsl.ActivatedModesModification(outer, dsl.Costs(),
			dsl.ModeOf("declared", dsl.ReturnToHand(inner), dsl.WithTargets(inner)),
			dsl.ModeOf("other mode read", dsl.ReturnToHand(inner))),
	} {
		card := dsl.NewInstant("Invalid modal grant", "", dsl.WithAbility(dsl.Spell(dsl.Until(core.Indefinite, invalid), dsl.WithTargets(outer))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("mode read a target from another scope")
		}
	}
}
