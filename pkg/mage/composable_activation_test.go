package mage

import (
	"fmt"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableLimitedPumpPayment(t *testing.T) {
	for _, limit := range []int{1, 2} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.turns.SetActivePlayerIndex(1)
			g.turns.SetStep(Upkeep)
			card := NewCreature("Source", "", 1, 1, WithAbility(Activate(
				Costs(ManaPayment("{R}")),
				Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(0))), ActivationLimit(limit))))
			g.PutOnBattlefield(card, a.PlayerID())
			activate := func(game *Game) error {
				return game.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil)
			}
			if err := activate(g); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation changed the stack", err)
			}
			a.ManaPool().Add(Red, limit+1)
			for range limit {
				if err := activate(g); err != nil {
					t.Fatal("failed payment consumed an activation", err)
				}
			}
			if err := activate(g); err == nil || g.StackSize() != limit || a.ManaPool().TotalMana() != 1 {
				t.Fatal("limit did not prevent payment before resolution", err)
			}
			clone := g.Clone()
			clone.ResolveStack()
			clone.ApplyContinuousEffects()
			if got := clone.FindPermanent(card.ID()).CurrentPower(clone); got != 1+limit {
				t.Fatalf("power = %d, want %d", got, 1+limit)
			}
			if err := activate(clone); err == nil {
				t.Fatal("resolution reset the activation limit")
			}
			clone.doCleanupActions()
			clone.turns.SetStep(Upkeep)
			clone.ApplyContinuousEffects()
			if got := clone.FindPermanent(card.ID()).CurrentPower(clone); got != 1 {
				t.Fatalf("cleanup retained the bonus: power = %d", got)
			}
			clone.GetPlayer(a.PlayerID()).ManaPool().Add(Red, 1)
			if err := activate(clone); err != nil {
				t.Fatal("cleanup retained the activation limit", err)
			}
			if err := activate(g); err == nil || g.StackSize() != limit || a.ManaPool().TotalMana() != 1 {
				t.Fatal("clone changed the original activation state", err)
			}
		})
	}
}

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
