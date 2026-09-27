package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEnchantmentEntrySubtypeUsesEntryCharacteristics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		subtype    string
		controller bool
		phased     bool
		want       int
	}{
		{"aura", "Aura", false, false, 1},
		{"other enchantment", "Saga", false, false, 0},
		{"opposing aura", "Aura", true, false, 0},
		{"phased watcher", "Aura", false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters().OfSubtype("Aura"), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			entry := NewEnchantment("Entry", "")
			entry.AddSubType(tc.subtype)
			entry.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.controller {
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

func TestEnchantmentEntrySubtypeCaptureKeepsIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered aura")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().OfSubtype("Aura").WithPermanent(entered),
		AddCountersTo(From(entered), core.P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.AddSubType("Aura")
	entry.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	stackCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if stackCopy == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.FindPermanent(entry.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("clone changed original: %d", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(entry, a.PlayerID())
	g.ResolveStack()
	if got := g.FindPermanent(entry.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("returned entry received %d counters", got)
	}
}

func TestEnchantmentEntrySubtypeSeesContinuousEffect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().OfSubtype("Aura"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "", WithAbility(Static(While(
		Controls(AnyPermanent), SetSubtypeFamily(This, SubtypeEnchantment, "Aura")))))
	entry.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("triggers = %d, want 1", got)
	}
}

func TestEnchantmentEntrySubtypeDeclaration(t *testing.T) {
	for _, pattern := range []controlledEnchantmentEntryPattern{
		EnchantmentYouControlEnters().OfSubtype(""),
		EnchantmentYouControlEnters().OfSubtype("Goblin"),
		EnchantmentYouControlEnters().OfSubtype("Aura").OfSubtype("Saga"),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid enchantment subtype filter")
		}
	}
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().OfSubtype("Aura"), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(EnchantmentYouControlEnters().OfSubtype("Aura")); got.Role != "event-pattern" || got.Text == "" {
		t.Fatalf("inspection = %+v", got)
	}
}
