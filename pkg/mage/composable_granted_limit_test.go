package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticGrantedLimit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), GainLifeFor(You, Fixed(2)), ActivationLimit(2),
	))))
	recipient, other := NewArtifact("Recipient", ""), NewArtifact("Other", "")
	for _, card := range []Card{donor, recipient, other} {
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	a.ManaPool().Add(Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	for range 3 {
		clone.ApplyContinuousEffects()
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone.ApplyContinuousEffects()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("reevaluation reset limit")
	}
	if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 2 || clone.StackSize() != 2 {
		t.Fatal("rejected activation changed payment or stack")
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal("recipients share limit", err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal("clone changed original count", err)
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("original limit not enforced")
	}
	clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
	clone.ApplyContinuousEffects()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 26 {
		t.Fatal("donor departure stopped pending abilities")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal("new turn retained limit", err)
	}
}

func TestStaticGrantedLimitIdentity(t *testing.T) {
	for _, change := range []string{"reevaluation", "control", "recipient phases", "donor phases", "recipient returns", "donor returns", "second donor"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			rule := GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))
			donor := NewEnchantment("Donor", "", WithAbility(Static(rule)))
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(recipient, a.PlayerID())
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			controller, index, allowed := a.PlayerID(), 0, false
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: donor.ID(), TargetID: recipient.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn})
				controller = b.PlayerID()
			case "recipient phases":
				g.PhaseOut(recipient.ID())
				g.ApplyContinuousEffects()
				g.PhaseIn([]uuid.UUID{recipient.ID()})
			case "donor phases":
				g.PhaseOut(donor.ID())
				g.ApplyContinuousEffects()
				g.PhaseIn([]uuid.UUID{donor.ID()})
			case "recipient returns":
				g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
				g.PutOnBattlefield(recipient, a.PlayerID())
				allowed = true
			case "donor returns":
				g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
				g.PutOnBattlefield(donor, a.PlayerID())
				allowed = true
			case "second donor":
				g.PutOnBattlefield(NewEnchantment("Second donor", "", WithAbility(Static(rule))), a.PlayerID())
				index, allowed = 1, true
			}
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) <= index {
				t.Fatal("missing grant after change")
			}
			err := g.ActivateAbilityByIndex(controller, recipient.ID(), index, nil)
			if (err == nil) != allowed {
				t.Fatalf("allowed = %v, error = %v", allowed, err)
			}
		})
	}
}

func TestStaticGrantedManaLimit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsLand), Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)), ActivationLimit(1),
	)))), a.PlayerID())
	land := NewLand("Recipient")
	g.PutOnBattlefield(land, a.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 1}); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || a.ManaPool().Count(Blue) != 1 {
		t.Fatal("mana did not resolve immediately")
	}
	a.ManaPool().Clear()
	g.MutablePermanent(land.ID()).Tapped = false
	g.ApplyContinuousEffects()
	if g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("planner ignored limit")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err == nil || g.FindPermanent(land.ID()).Tapped {
		t.Fatal("blocked mana activation paid cost", err)
	}
	g.doCleanupActions()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(land.ID()).Tapped = false
	g.ApplyContinuousEffects()
	a.ManaPool().Clear()
	if g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("manual mana activation did not record limit")
	}
}

func TestStaticGrantedLimitValidation(t *testing.T) {
	for _, options := range [][]GrantedActivationOption{{ActivationLimit(0)}, {ActivationLimit(-1)}, {ActivationLimit(1), ActivationLimit(2)}} {
		err := validateFactory("Invalid limit", func() Card {
			return NewEnchantment("Invalid limit", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), Regenerate(This), options...))))
		})
		if err == nil {
			t.Fatal("invalid limit accepted")
		}
	}
}

func TestStaticGrantedLimitCopiesAndDistinctGrants(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	rule := GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), GainLifeFor(target, Fixed(1)), WithTargets(target), ActivationLimit(2))
	donor := NewEnchantment("Donor", "", WithAbility(Static(rule, rule)))
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(donor, a.PlayerID())
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("missing target accepted")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.GetStack().Peek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal("copy consumed activation", err)
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err == nil {
		t.Fatal("limit ignored")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal("separate grants share count", err)
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 21 {
		t.Fatalf("unexpected life: %d, %d", a.Life(), b.Life())
	}
}
