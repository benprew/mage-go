package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestActionTargetSurchargeContract(t *testing.T) {
	for _, route := range []string{"hand", "alternate"} {
		for count := 0; count <= 3; count++ {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("recipients")
			action := NewAction(ActionSpell, WithEffects(GainLifeFor(You, Fixed(1))), WithTarget(targets), WithAdditionalManaPerTargetBeyondFirst(2))
			spell := NewInstant("Action surcharge", "{X}{R}", WithAbility(action))
			spell.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			ids := []uuid.UUID{a.PlayerID(), b.PlayerID(), victim.ID()}[:count]
			a.ManaPool().Add(Red, 1)
			a.ManaPool().Add(Colorless, 3+2*max(0, count-1))
			var err error
			if route == "hand" {
				a.AddToHand(spell)
				err = g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 3)
			} else {
				a.AddToGraveyard(spell)
				err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), ZoneGraveyard, spell.ManaCost(), ids, 3)
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != 3 {
				t.Fatal("action option changed target count or payment")
			}
		}
	}
	for _, amount := range []int{-1, 0} {
		spell := NewInstant("Invalid", "", WithAbility(NewAction(ActionSpell, WithEffects(GainLifeFor(You, Fixed(1))), WithAdditionalManaPerTargetBeyondFirst(amount))))
		if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
}

func TestActionTargetSurchargeRejectsActivation(t *testing.T) {
	card := NewArtifact("Invalid activation", "", WithAbility(NewAction(ActionActivated, WithEffects(GainLifeFor(You, Fixed(1))), WithAdditionalManaPerTargetBeyondFirst(1))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("activation accepted a spell announcement cost")
	}
}

func TestActionTargetSurchargeGroupsAndLockedCost(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	first, second := DeclarePlayerTarget("first"), DeclarePlayerTarget("second")
	action := NewAction(ActionSpell, WithEffects(GainLifeFor(You, Fixed(1))), WithTarget(first), WithTarget(second), WithAdditionalManaPerTargetBeyondFirst(2))
	reducer := g.PutOnBattlefield(NewCreature("Reducer", "", 1, 1), a.PlayerID())
	spell := NewInstant("Locked action", "{X}{R}", WithAbility(action), WithAdditionalCost(SacrificeNMatchingCost(1, IsCreature, "Sacrifice a creature")), WithSelfCostReduction(FixedAmount(1), CondControlsMatching(IsCreature)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	g.effects.Rules.SpellCostIncreases[Red] = 1
	g.effects.Rules.SpellCostReductions[Red] = 2
	a.ManaPool().Add(Red, 1)
	a.ManaPool().Add(Colorless, 4)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID(), b.PlayerID()}, 4); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || g.FindPermanent(reducer.ID()) != nil || g.StackPeek().XValue != 4 {
		t.Fatal("cost was not locked before sacrifice")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("action did not resolve")
	}
}
