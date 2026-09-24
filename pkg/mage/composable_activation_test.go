package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableActivationRestrictions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.turns.SetStep(Upkeep)
	untapOnly := Activate(Costs(), GainLifeFor(You, Fixed(1)), OnlyDuring(Untap))
	if untapOnly.CanActivate(a.PlayerID(), g) {
		t.Fatal("zero-valued untap restriction allowed upkeep activation")
	}
	ability := Activate(Costs(), GainLifeFor(You, Fixed(1)), OnlyDuring(Upkeep), ActivationLimit(2), ActivationIf(Controls(IsCreature)))
	card := NewCreature("Source", "", 1, 1, WithAbility(ability))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	ability.SetSource(card.ID())
	g.turns.SetStep(Upkeep)
	if !ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("valid activation rejected")
	}
	g.turns.SetStep(EndStep)
	if ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("step restriction ignored")
	}
	g.turns.SetStep(Upkeep)
	ability.activationsThisTurn = 2
	if ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("limit ignored")
	}
	ability.activationsThisTurn = 0
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("state condition ignored")
	}
}

func TestComposableActivationOptionValidation(t *testing.T) {
	for _, options := range [][]ActivationOption{
		{OnlyAsSorcery(), OnlyDuring(Upkeep)},
		{OnlyDuring(Upkeep), OnlyDuring(Upkeep)},
		{OnlyOnYourTurn(), OnlyOnYourTurn()},
		{OnlyDuring(Upkeep), OnlyOnYourTurn(), OnlyDuring(EndStep)},
		{OnlyOnYourTurn(), OnlyDuring(Upkeep), OnlyOnYourTurn()},
		{OnlyOnYourTurn(), OnlyAsSorcery()},
		{ActivationLimit(1), ActivationLimit(2)},
		{ActivationLimit(0)},
		{OnlyDuring(PhaseStep(-1))},
		{ActivationIf(nil)},
	} {
		if err := validateFactory("Invalid activation", func() Card {
			return NewCreature("Invalid activation", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), options...)))
		}); err == nil {
			t.Fatal("invalid options accepted")
		}
	}
	ability := Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(AllConditions()))
	if err := ValidateCardDefinition("Invalid condition", NewCreature("Invalid condition", "", 1, 1, WithAbility(ability))); err == nil {
		t.Fatal("activation condition was not inspected")
	}
}
