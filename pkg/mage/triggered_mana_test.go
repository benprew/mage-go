package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTappedForManaTriggerResolvesImmediately(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	producer := BindPlayer("producer")
	watcher := NewEnchantment("Bonus", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(producer),
		AddManaToPlayer(FromPlayer(producer), Green, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Ordinary", "", WithAbility(WhenEvent(
		LandTappedForMana(), GainLifeFor(You, Fixed(1))))), b.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(Blue)), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := a.ManaPool().Count(Green); got != 1 {
		t.Fatalf("bonus mana = %d, want 1 before priority", got)
	}
	if g.StackSize() != 0 || b.Life() != 20 {
		t.Fatalf("stack = %d, life = %d", g.StackSize(), b.Life())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("ordinary trigger count = %d, want 1", g.StackSize())
	}
}

func TestTappedForManaTriggerLeavesOtherPendingTriggers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Earlier", "", WithAbility(NewTriggered(
		EvtDrawStep, false, GainLifeFor(You, Fixed(1))))), b.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Bonus", "", WithAbility(WhenEvent(
		LandTappedForMana(), AddManaAmount(Red, Fixed(1))))), b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDrawStep, PlayerID: a.PlayerID()})
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(Blue)), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(Red) != 1 || b.Life() != 20 {
		t.Fatalf("red = %d, life = %d, stack = %d", b.ManaPool().Count(Red), b.Life(), g.StackSize())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending trigger count = %d, want 1", g.StackSize())
	}
}

func TestTappedForManaTriggerKeepsNewOrdinaryTriggerPending(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Bonus", "", WithAbility(WhenEvent(
		LandTappedForMana(), Sequence(GainLifeFor(You, Fixed(1)), AddManaAmount(Red, Fixed(1)))))), b.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(
		EvtLifeGained, false, GainLifeFor(You, Fixed(1))))), b.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(Blue)), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(Red) != 1 || b.Life() != 21 || g.StackSize() != 0 {
		t.Fatalf("red = %d, life = %d, stack = %d", b.ManaPool().Count(Red), b.Life(), g.StackSize())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("ordinary trigger count = %d, want 1", g.StackSize())
	}
}

func TestTappedForManaTriggerPhasingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := g.PutOnBattlefield(NewEnchantment("Bonus", "", WithAbility(WhenEvent(
		LandTappedForMana(), AddManaAmount(Green, Fixed(1))))), b.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(Blue)), a.PlayerID())
	clone := g.Clone()
	clone.MutablePermanent(watcher.ID()).PhasedOut = true
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(b.PlayerID()).ManaPool().Count(Green) != 0 || b.ManaPool().Count(Green) != 0 {
		t.Fatal("phased or original watcher produced mana")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(Green) != 1 || clone.GetPlayer(b.PlayerID()).ManaPool().Count(Green) != 0 {
		t.Fatal("immediate mana crossed game clone")
	}
}

func TestTappedForManaTriggerWithTargetUsesStack(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	watcher := NewEnchantment("Targeted", "", WithAbility(WhenEvent(
		LandTappedForMana(), AddManaToPlayer(target, Green, Fixed(1)), WithTargets(target))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, b.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(Blue)), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(Green) != 0 || g.StackSize() != 0 {
		t.Fatal("targeted trigger resolved immediately")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("targeted trigger count = %d, want 1", g.StackSize())
	}
}
