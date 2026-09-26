package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLandEntryActorDiffersFromController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("entry actor")
	controller := BindPlayer("entry controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().OfSubtype("Swamp").WithActor(actor).WithController(controller),
		Sequence(GainLifeFor(FromPlayer(actor), Fixed(2)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Swamp", WithSubTypes("Swamp"))
	land.SetOwner(b.PlayerID())
	g.PutOnBattlefieldBy(land, b.PlayerID(), a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefieldBy(land, a.PlayerID(), b.PlayerID())
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 24 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}

func TestLandEntryActorDeclaration(t *testing.T) {
	actor := BindPlayer("actor")
	for _, pattern := range []landEntersPattern{
		LandEnters().WithActor(nil),
		LandEnters().WithActor(BindPlayer("")),
		LandEnters().WithActor(actor).WithActor(actor),
		LandEnters().WithActor(actor).WithController(actor),
	} {
		card := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid actor declaration")
		}
	}
	card := NewArtifact("Unbound", "", WithAbility(WhenEvent(LandEnters(), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted unbound actor read")
	}
}

func TestLandEntryActorRequiresKnownEntryPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().WithActor(actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(NewLand("Setup land"), b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("entry without a known actor triggered")
	}
	g.SetStep(core.PrecombatMain)
	land := NewLand("Played land")
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	if err := g.playLandCore(a.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 {
		t.Fatalf("played land stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}
