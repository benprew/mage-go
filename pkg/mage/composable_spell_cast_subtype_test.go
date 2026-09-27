package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastSubtypeUsesCastStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		SpellCast().OfType(TypeEnchantment).OfEnchantmentSubtype("Aura").ByOpponent(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewAura("Aura spell", "")
	spell.SetOwner(b.PlayerID())
	other := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()}
	g.pushStack(other)
	current := &StackObject{ID: uuid.New(), Card: NewEnchantment("Current form", ""), SourceID: spell.ID(), Controller: b.PlayerID()}
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: current.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("stack size=%d, want 2", got)
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: other.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 3 {
		t.Fatalf("stack size=%d, want 3", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("original life after clone resolution=%d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("life=%d, want 21", got)
	}
}

func TestSpellCastSubtypeSourceLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		phase bool
		leave bool
		want  int
	}{
		{name: "present", want: 1},
		{name: "phased", phase: true},
		{name: "left", leave: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfEnchantmentSubtype("Aura"), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			if tc.leave {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			}
			spell := NewAura("Aura", "")
			spell.SetOwner(b.PlayerID())
			object := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()}
			g.pushStack(object)
			g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: object.ID, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if got := g.StackSize() - 1; got != tc.want {
				t.Fatalf("triggers=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestSpellCastSubtypeDeclaration(t *testing.T) {
	pattern := SpellCast().OfEnchantmentSubtype("Aura")
	if got := InspectDefinition(pattern); got.Role != "event-pattern" || !strings.Contains(got.Text, "Aura") {
		t.Fatalf("inspection=%+v", got)
	}
	for _, invalid := range []EventPattern{
		SpellCast().OfEnchantmentSubtype(""),
		SpellCast().OfEnchantmentSubtype("Bogus"),
		SpellCast().OfEnchantmentSubtype("Aura").OfEnchantmentSubtype("Aura"),
		SpellCast().OfType(TypeCreature).OfEnchantmentSubtype("Aura"),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(invalid, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "subtype") {
			t.Fatalf("validation=%v", err)
		}
	}
}
