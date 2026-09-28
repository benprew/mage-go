package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestCreatureEntryControllerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("entrant controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(2)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entrant := NewCreature("Entrant", "", 1, 1)
	entrant.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entrant, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(entrant.ID()))
	g.PutOnBattlefield(entrant, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 24 {
		t.Fatalf("life A=%d B=%d, want 20 and 24", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 24 {
		t.Fatal("clone lost captured controller")
	}
}

func TestCreatureEntryControllerCapturePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("entrant controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PhaseOut(watcher.ID())
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn([]uuid.UUID{watcher.ID()})
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.PutTriggersOnStack()
	g.PhaseOut(watcher.ID())
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatalf("life=%d, want 21", b.Life())
	}
}

func TestCreatureEntryControllerCaptureValidation(t *testing.T) {
	for _, pattern := range []creatureEntersPattern{
		AnotherCreatureEnters().WithController(nil),
		AnotherCreatureEnters().WithController(BindPlayer("")),
		AnotherCreatureEnters().WithController(BindPlayer("one")).WithController(BindPlayer("two")),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid pattern: %+v", pattern)
		}
	}
	missing := BindPlayer("missing")
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureEnters(), GainLifeFor(FromPlayer(missing), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted undominated binding")
	}
}
