package mage

import "testing"

func TestSourceOnlyDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	owner, controller := BindPlayer("owner"), BindPlayer("controller")
	card := NewEnchantment("Source", "", WithAbility(WhenEvent(PermanentDies().SourceOnly().WithOwner(owner).WithController(controller), Sequence(GainLifeFor(FromPlayer(owner), Fixed(2)), GainLifeFor(FromPlayer(controller), Fixed(3))))))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Source", card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, b.PlayerID())
	otherCard := NewArtifact("Other", "")
	otherCard.SetOwner(b.PlayerID())
	other := g.PutOnBattlefield(otherCard, b.PlayerID())
	g.DestroyPermanent(other)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unrelated death matched")
	}
	g.DestroyPermanent(p)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("source death missing")
	}
	clone := g.Clone()
	a.RemoveFromGraveyard(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 23 {
		t.Fatal("departure bindings changed on return")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone changed")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
		t.Fatal("clone lost bindings")
	}
}

func TestSourceOnlyDeathInvalid(t *testing.T) {
	for _, pattern := range []creatureDiesPattern{PermanentDies().SourceOnly().SourceOnly(), AnotherCreatureDies().SourceOnly()} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid source restriction accepted")
		}
	}
}
