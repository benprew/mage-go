package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLandTappedForManaPlayerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("mana player")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player), DealDamageFrom(This, FromPlayer(player), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(1)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatal("clone changed original or lost event player")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 19 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestLandTappedForManaDeclaration(t *testing.T) {
	player := BindPlayer("mana player")
	for _, pattern := range []landTappedForManaPattern{
		LandTappedForMana().WithPlayer(nil),
		LandTappedForMana().WithPlayer(BindPlayer("")),
		LandTappedForMana().WithPlayer(player).WithPlayer(player),
	} {
		card := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid event binding")
		}
	}
	card := NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
		GainLifeFor(FromPlayer(player), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted read without event binding")
	}
}

func TestLandTappedForManaFilteringAndSourceLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("player")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player), GainLifeFor(FromPlayer(player), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact mana", "", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(1)))))
	g.PutOnBattlefield(artifact, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	zero := NewLand("Zero mana", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(0)))))
	g.PutOnBattlefield(zero, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), zero.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("nonland or zero production matched")
	}
	g.PhaseOut(watcher.ID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Green, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed mana event")
	}
	g.PhaseIn([]uuid.UUID{watcher.ID()})
	second := NewLand("Second", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Green, Fixed(1)))))
	g.PutOnBattlefield(second, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}
