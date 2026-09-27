package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureDeathSubtypeEventState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	goblin := NewCreature("Goblin", "", 2, 2, WithSubTypes("Goblin"))
	goblin.SetOwner(b.PlayerID())
	elf := NewCreature("Elf", "", 2, 2, WithSubTypes("Elf"))
	elf.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(goblin, b.PlayerID())
	g.PutOnBattlefield(elf, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(goblin.ID()), g.FindPermanent(elf.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want one Goblin death trigger", g.StackSize())
	}
	clone := g.Clone()
	g.zones.ClearLKI()
	g.PutOnBattlefield(goblin, b.PlayerID())
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 21 || b.Life() != 20 {
		t.Fatal("death trigger lost event subtype or clone isolation")
	}
}

func TestCreatureDeathSubtypePhasedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	goblin := NewCreature("Goblin", "", 2, 2, WithSubTypes("Goblin"))
	goblin.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(goblin, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(goblin.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased-out watcher observed a death")
	}
}

func TestCreatureDeathSubtypeUsesEffectiveType(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2, WithSubTypes("Elf"))
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetSubtypeFamily(This, SubtypeCreature, "Goblin")), victim.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want one trigger from effective subtype", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
}

func TestCreatureDeathSubtypeDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pattern creatureDiesPattern
		want    string
	}{
		{"valid", CreatureDies().OfSubtype("Goblin"), ""},
		{"empty", CreatureDies().OfSubtype(""), "subtype"},
		{"wrong family", CreatureDies().OfSubtype("Island"), "creature subtype"},
		{"duplicate", CreatureDies().OfSubtype("Goblin").OfSubtype("Elf"), "one subtype"},
		{"permanent", PermanentDies().OfType(core.TypeLand).OfSubtype("Goblin"), "creature"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(tc.pattern, GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation = %v, want %q", err, tc.want)
			}
		})
	}
}
