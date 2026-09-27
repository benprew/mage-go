package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPermanentSubtypeBranchEntryAndDeparture(t *testing.T) {
	for _, tc := range []struct {
		name, subtype            string
		depart, returned, phased bool
		want                     int
	}{
		{"aura", "Aura", false, false, false, 22},
		{"saga", "Saga", false, false, false, 20},
		{"departed aura", "Aura", true, false, false, 22},
		{"returned aura", "Aura", true, true, false, 22},
		{"phased aura", "Aura", false, false, true, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			entered := BindPermanent("entered")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters().WithPermanent(entered),
				IfPermanentHasEnchantmentSubtype(From(entered), "Aura", GainLifeFor(You, Fixed(2))))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			entry := NewEnchantment("Entry", "")
			entry.AddSubType(tc.subtype)
			entry.SetOwner(b.PlayerID())
			g.PutOnBattlefield(entry, a.PlayerID())
			g.PutTriggersOnStack()
			if tc.depart {
				g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
			}
			if tc.returned {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
				g.PutOnBattlefield(entry, a.PlayerID())
			}
			if tc.phased {
				g.PhaseOut(entry.ID())
			}
			g.ResolveStack()
			if got := a.Life(); got != tc.want {
				t.Fatalf("life = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPermanentSubtypeBranchDepartureKeepsEffectiveSubtype(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		IfPermanentHasEnchantmentSubtype(From(entered), "Aura", GainLifeFor(You, Fixed(2))))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "", WithAbility(Static(While(
		Controls(AnyPermanent), SetSubtypeFamily(This, SubtypeEnchantment, "Aura")))))
	entry.AddSubType("Saga")
	entry.SetOwner(b.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("life after effective Aura departure = %d", got)
	}
}

func TestPermanentSubtypeBranchDeclaration(t *testing.T) {
	bound := BindPermanent("entered")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(bound),
		IfPermanentHasEnchantmentSubtype(From(bound), "Aura", GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{IfPermanentHasEnchantmentSubtype(nil, "Aura", GainLifeFor(You, Fixed(1))), "permanent"},
		{IfPermanentHasEnchantmentSubtype(From(bound), "Aura", GainLifeFor(You, Fixed(1))), "producer"},
		{IfPermanentHasEnchantmentSubtype(This, "", GainLifeFor(You, Fixed(1))), "subtype"},
		{IfPermanentHasEnchantmentSubtype(This, "Goblin", GainLifeFor(You, Fixed(1))), "subtype"},
		{IfPermanentHasEnchantmentSubtype(This, "Aura", nil), "then"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestPermanentSubtypeBranchCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		IfPermanentHasEnchantmentSubtype(From(entered), "Aura", GainLifeFor(You, Fixed(2))))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.AddSubType("Aura")
	entry.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("clone changed original life: %d", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone controller life = %d", got)
	}
	if got := clone.GetPlayer(b.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone copy controller life = %d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("original controller life = %d", got)
	}
	if got := b.Life(); got != 22 {
		t.Fatalf("original copy controller life = %d", got)
	}
}

func TestPermanentSubtypeBranchTargetDoesNotUseDepartureLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("enchantment", IsEnchantment)
	ability := Spell(IfPermanentHasEnchantmentSubtype(target, "Aura", GainLifeFor(You, Fixed(2))), WithTargets(target))
	spell := NewInstant("Check", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.AddSubType("Aura")
	entry.SetOwner(b.PlayerID())
	g.PutOnBattlefield(entry, b.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{entry.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
	g.PutOnBattlefield(entry, b.PlayerID())
	g.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("illegal target gained life: %d", got)
	}
}
