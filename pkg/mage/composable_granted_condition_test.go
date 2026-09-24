package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticGrantedCondition(t *testing.T) {
	for _, scenario := range []string{"allowed", "missing source counters", "missing controlled creature"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), GainLifeFor(You, Fixed(2)),
				ActivationIf(SourceHasCounters(Charge, 1)), ActivationIf(Controls(IsCreature)),
			))))
			g.PutOnBattlefield(donor, a.PlayerID())
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(recipient, b.PlayerID())
			other := NewArtifact("Other recipient", "")
			g.PutOnBattlefield(other, b.PlayerID())
			if scenario != "missing source counters" {
				g.MutablePermanent(recipient.ID()).AddCounter(Charge, 1)
			} else {
				g.MutablePermanent(donor.ID()).AddCounter(Charge, 1)
			}
			creature := NewCreature("Creature", "", 1, 1)
			controller := b.PlayerID()
			if scenario == "missing controlled creature" {
				controller = a.PlayerID()
			}
			g.PutOnBattlefield(creature, controller)
			g.ApplyContinuousEffects()
			b.ManaPool().Add(Colorless, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err == nil {
				t.Fatal("recipient without counters activated")
			}
			err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
			if scenario != "allowed" {
				if err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
					t.Fatal("rejected activation changed payment or stack", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || b.Life() != 20 {
				t.Fatal("incorrect payment or resolution timing")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(creature.ID()))
			clone.ApplyContinuousEffects()
			clone.GetPlayer(b.PlayerID()).ManaPool().Add(Colorless, 1)
			if err := clone.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
				t.Fatal("clone used stale condition state")
			}
			if clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 1 || clone.StackSize() != 1 {
				t.Fatal("failed clone activation changed payment or stack")
			}
			clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("resolution rechecked condition or used donor controller")
			}
			if b.Life() != 20 || g.StackSize() != 1 || g.FindPermanent(creature.ID()) == nil {
				t.Fatal("clone changed original state")
			}
			b.ManaPool().Add(Colorless, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
				t.Fatal("original lost its valid condition", err)
			}
		})
	}
}

func TestStaticGrantedConditionValidation(t *testing.T) {
	for _, condition := range []StateCondition{nil, AllConditions(), SourceHasCounters(Charge, -1)} {
		err := validateFactory("Invalid grant condition", func() Card {
			return NewEnchantment("Invalid grant condition", "", WithAbility(Static(GrantEachActivatedAbility(
				MatchingPermanents(IsArtifact), Costs(), Regenerate(This), ActivationIf(condition),
			))))
		})
		if err == nil {
			t.Fatal("invalid condition accepted")
		}
		if condition != nil && !strings.Contains(err.Error(), "restrictions[0].condition") {
			t.Fatalf("missing condition path: %v", err)
		}
	}
}

func TestStaticGrantedManaCondition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewEnchantment("Mana donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsLand), Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)),
		ActivationIf(SourceHasCounters(Charge, 1)),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewLand("Recipient")
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.MutablePermanent(donor.ID()).AddCounter(Charge, 1)
	g.ApplyContinuousEffects()
	if g.CanAfford(b.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("planner ignored recipient condition")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil || g.FindPermanent(recipient.ID()).Tapped {
		t.Fatal("rejected mana activation paid tap cost", err)
	}
	clone := g.Clone()
	clone.MutablePermanent(recipient.ID()).AddCounter(Charge, 1)
	clone.ApplyContinuousEffects()
	if !clone.CanAfford(b.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("planner did not read current recipient condition")
	}
	if err := clone.AutoTapForCost(b.PlayerID(), ManaCost{Blue: 1}); err != nil {
		t.Fatal(err)
	}
	if clone.StackSize() != 0 || clone.GetPlayer(b.PlayerID()).ManaPool().Count(Blue) != 1 || !clone.FindPermanent(recipient.ID()).Tapped {
		t.Fatal("granted mana did not resolve immediately")
	}
	if g.FindPermanent(recipient.ID()).Tapped || b.ManaPool().TotalMana() != 0 || g.CanAfford(b.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("clone changed original condition or payment")
	}
}
