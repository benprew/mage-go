package mage

import (
	"strings"
	"testing"
)

func TestCreatureDiesOwnerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	owner, controller := BindPlayer("owner"), BindPlayer("controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithOwner(owner).WithController(controller),
		Sequence(GainLifeFor(FromPlayer(owner), Fixed(2)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	first.SetOwner(b.PlayerID())
	second.SetOwner(a.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 24 {
		t.Fatalf("life=(%d,%d), want (25,24)", a.Life(), b.Life())
	}
}

func TestCreatureDiesOwnerCaptureSourceAndPhasing(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		owner := BindPlayer("owner")
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
			CreatureDies().WithOwner(owner), GainLifeFor(FromPlayer(owner), Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		victim := NewCreature("Victim", "", 1, 1)
		victim.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(victim, a.PlayerID())
		if phase {
			g.PhaseOut(watcher.ID())
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phased source observed death")
			}
			continue
		}
		g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatalf("stack=%d, want 2", g.StackSize())
		}
		g.ResolveStack()
		if a.Life() != 21 || b.Life() != 21 {
			t.Fatalf("life=(%d,%d), want (21,21)", a.Life(), b.Life())
		}
	}
}

func TestCreatureDiesOwnerCaptureValidation(t *testing.T) {
	owner := BindPlayer("owner")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		body    ResolvingEffect
		want    string
	}{
		{"nil", CreatureDies().WithOwner(nil), GainLifeFor(You, Fixed(1)), "binding"},
		{"empty", CreatureDies().WithOwner(BindPlayer("")), GainLifeFor(You, Fixed(1)), "binding name"},
		{"unbound", CreatureDies(), GainLifeFor(FromPlayer(owner), Fixed(1)), "producer"},
		{"duplicate", CreatureDies().WithOwner(owner).WithController(owner), GainLifeFor(You, Fixed(1)), "different bindings"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(tc.pattern, tc.body)))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation=%v, want %q", err, tc.want)
			}
		})
	}
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().Nontoken().WithOwner(owner), GainLifeFor(FromPlayer(owner), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}
