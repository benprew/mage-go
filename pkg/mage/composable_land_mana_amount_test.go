package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLandTappedForManaAmountCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("produced")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithAmount(amount), GainLifeFor(You, FromNumber(amount)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(2)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone lost amount or changed original")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestLandTappedForManaAmountDeclaration(t *testing.T) {
	amount := BindNumber("produced")
	for _, pattern := range []landTappedForManaPattern{
		LandTappedForMana().WithAmount(nil),
		LandTappedForMana().WithAmount(BindNumber("")),
		LandTappedForMana().WithAmount(amount).WithAmount(amount),
	} {
		card := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid amount binding")
		}
	}
	card := NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
		GainLifeFor(You, FromNumber(amount)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted read without event binding")
	}
}
