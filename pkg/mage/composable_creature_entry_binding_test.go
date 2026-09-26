package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureEntryPermanentBindingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		controllerB bool
		depart      bool
		phase       bool
		want        int
	}{
		{"controlled", false, false, false, 1},
		{"opponent controlled", true, false, false, 0},
		{"returned", false, true, false, 0},
		{"phased", false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			entered := BindPermanent("entered")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().WithPermanent(entered),
				AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			creature := NewCreature("Creature", "", 2, 2)
			creature.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.controllerB {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(creature, controller)
			g.PutTriggersOnStack()
			if tc.controllerB {
				if g.StackSize() != 0 {
					t.Fatal("opponent entry triggered")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("controlled entry did not trigger")
			}
			if tc.depart {
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
				g.PutOnBattlefield(creature, a.PlayerID())
			}
			if tc.phase {
				g.PhaseOut(creature.ID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if !tc.phase {
				if got := g.FindPermanent(creature.ID()).Counters[core.P1P1]; got != 0 {
					t.Fatalf("clone changed original: %d counters", got)
				}
			}
			g.ResolveStack()
			if tc.phase {
				g.PhaseIn([]uuid.UUID{creature.ID()})
			}
			if got := g.FindPermanent(creature.ID()).Counters[core.P1P1]; int(got) != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestCreatureEntryPermanentBindingDeclaration(t *testing.T) {
	entered := BindPermanent("entered")
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().WithPermanent(entered), AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(good.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || len(meta.Children[0].Children) == 0 {
		t.Fatalf("entry binding missing from inspection: %+v", meta)
	}
	for _, pattern := range []creatureEntersPattern{
		AnotherCreatureYouControlEnters().WithPermanent(nil),
		AnotherCreatureYouControlEnters().WithPermanent(BindPermanent("")),
	} {
		bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid entry binding")
		}
	}
	unbound := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters(), AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound event read")
	}
}

func TestCreatureEntryPermanentBindingStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().WithPermanent(entered),
		AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	creature := NewCreature("Creature", "", 2, 2)
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("entry did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.ResolveStack()
	if got := g.FindPermanent(creature.ID()).Counters[core.P1P1]; got != 2 {
		t.Fatalf("copies used different entered objects: %d counters", got)
	}
}
