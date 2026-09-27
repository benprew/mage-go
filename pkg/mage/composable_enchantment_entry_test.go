package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEnchantmentYouControlEntersCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered enchantment")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	enchantment := NewEnchantment("Entry", "")
	enchantment.SetOwner(b.PlayerID())
	g.PutOnBattlefield(enchantment, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.FindPermanent(enchantment.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("clone changed original: %d counters", got)
	}
	g.ResolveStack()
	if got := g.FindPermanent(enchantment.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("entry added %d counters", got)
	}
}

func TestEnchantmentYouControlEntersFilters(t *testing.T) {
	for _, tc := range []struct {
		name     string
		opponent bool
		artifact bool
		phased   bool
		want     int
	}{
		{"controlled", false, false, false, 1},
		{"opponent controlled", true, false, false, 0},
		{"artifact", false, true, false, 0},
		{"phased source", false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters(), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			entry := NewEnchantment("Entry", "")
			if tc.artifact {
				entry = NewArtifact("Entry", "")
			}
			entry.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(entry, controller)
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEnchantmentYouControlEntersDeclaration(t *testing.T) {
	entered := BindPermanent("entered")
	for _, pattern := range []controlledEnchantmentEntryPattern{
		EnchantmentYouControlEnters().WithPermanent(nil),
		EnchantmentYouControlEnters().WithPermanent(BindPermanent("")),
		EnchantmentYouControlEnters().WithPermanent(entered).WithPermanent(entered),
	} {
		bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid capture")
		}
	}
	bad := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters(), AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("accepted unbound read")
	}
}

func TestEnchantmentYouControlEntersReturnedAndCopied(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		entered := BindPermanent("entered")
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
			EnchantmentYouControlEnters().WithPermanent(entered),
			AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		entry := NewEnchantment("Entry", "")
		entry.SetOwner(b.PlayerID())
		g.PutOnBattlefield(entry, a.PlayerID())
		g.PutTriggersOnStack()
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
			g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			g.PutOnBattlefield(entry, a.PlayerID())
		}
		clone := g.Clone()
		clone.ResolveStack()
		if got := g.FindPermanent(entry.ID()).Counters[core.P1P1]; got != 0 {
			t.Fatal("clone changed original")
		}
		g.ResolveStack()
		want := 2
		if returned {
			want = 0
		}
		if got := int(g.FindPermanent(entry.ID()).Counters[core.P1P1]); got != want {
			t.Fatalf("returned=%v: counters=%d, want %d", returned, got, want)
		}
	}
}

func TestEnchantmentYouControlEntersPhasedRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.SetOwner(b.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	g.PhaseOut(entry.ID())
	g.ResolveStack()
	g.PhaseIn([]uuid.UUID{entry.ID()})
	if got := g.FindPermanent(entry.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("phased entry received %d counters", got)
	}
}
