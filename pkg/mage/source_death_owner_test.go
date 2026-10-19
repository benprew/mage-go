package mage

import "testing"

func TestSourceDeathOwnerDeclaration(t *testing.T) {
	owner, other := BindPlayer("owner"), BindPlayer("other")
	for _, pattern := range []creatureDiesPattern{
		PermanentDies().SourceOnly().WithOwner(nil),
		PermanentDies().SourceOnly().WithOwner(owner).WithOwner(owner),
		PermanentDies().SourceOnly().WithOwner(owner).WithOwner(other),
		PermanentDies().SourceOnly().WithOwner(owner).WithController(owner),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid self-death owner declaration accepted")
		}
	}
}

func TestSourceDeathOwnerSavedFacts(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	owner := BindPlayer("owner")
	card := NewCreature("Subject", "", 1, 1, WithAbility(WhenEvent(PermanentDies().SourceOnly().WithOwner(owner), GainLifeFor(FromPlayer(owner), Fixed(3)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	g.DestroyPermanent(p)
	if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("death executed its body during detection")
	}
	clone := g.Clone()
	a.RemoveFromGraveyard(card.ID())
	card.SetOwner(b.PlayerID())
	g.PutOnBattlefield(card, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("wrong pending controller")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 20 {
		t.Fatal("return or copy replaced the saved owner")
	}
	if len(g.resolution.playerBindings) != 0 {
		t.Fatal("owner escaped the resolution frame")
	}
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("clone shared pending facts")
	}
}
