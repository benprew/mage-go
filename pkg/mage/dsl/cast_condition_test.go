package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCastConditionPublicDefinition(t *testing.T) {
	card := dsl.NewInstant("Cast condition test", "", dsl.Spell(
		dsl.IfCast(dsl.ControlledCreatureSubtypeAtCast("Dragon"), dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestRevealedCastConditionPublicDefinition(t *testing.T) {
	paid := dsl.BindNumber("reveal paid")
	revealed := dsl.BindCard("revealed Dragon")
	card := dsl.NewInstant("Reveal condition test", "", dsl.WithAbility(dsl.Spell(
		dsl.IfCast(dsl.RevealedSubtypeAtCast("Dragon"), dsl.GainLife(dsl.You, dsl.Fixed(1))))),
		dsl.WithAdditionalCost(dsl.OptionalRevealFromHandCost(
			dsl.HasSubTypeCardFilter("Dragon"), "Reveal a Dragon card", "Reveal?", paid, revealed)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestAnyCastPublicDefinition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	controller := dsl.BindPlayer("creature controller")
	card := dsl.NewInstant("Dragon condition test", "", dsl.Spell(dsl.Sequence(
		dsl.SnapshotPlayer(dsl.ControllerOf(target), controller),
		dsl.DealDamageFrom(dsl.This, target, dsl.Fixed(3)),
		dsl.IfCast(dsl.AnyCast(dsl.RevealedSubtypeAtCast("Dragon"),
			dsl.ControlledCreatureSubtypeAtCast("Dragon")),
			dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.Fixed(3))),
	), dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
