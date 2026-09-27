package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastWithXInManaCost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		SpellCast().WithXInManaCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, tc := range []struct {
		cost string
		x    int
	}{
		{"{X}{R}", 0},
		{"{X}{X}{U}", 2},
		{"{2}{R}", 0},
	} {
		spell := NewInstant("Spell", tc.cost)
		obj := &StackObject{ID: spell.ID(), SourceID: spell.ID(), Card: spell, XValue: tc.x, Controller: b.PlayerID()}
		g.stack.Push(obj)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
		g.stack.RemoveByID(obj.ID)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 22 {
			t.Fatalf("life = %d, want 22", got)
		}
	}
}

func TestSpellCastWithXInManaCostSourceLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		SpellCast().WithXInManaCost(), GainLifeFor(You, Fixed(1)))))
	permanent := g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "{X}{R}")
	obj := &StackObject{ID: spell.ID(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID()}
	g.stack.Push(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.stack.RemoveByID(obj.ID)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

func TestSpellCastWithXInManaCostDeclaration(t *testing.T) {
	pattern := SpellCast().WithXInManaCost()
	if got := InspectDefinition(pattern); got.Role != "event-pattern" {
		t.Fatalf("inspection = %#v", got)
	}
	valid := NewCreature("Valid", "", 2, 2, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		NewInstant("Wrong zone", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
		NewCreature("Duplicate", "", 2, 2, WithAbility(WhenEvent(pattern.WithXInManaCost(), GainLifeFor(You, Fixed(1))))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "spell cast") && !strings.Contains(err.Error(), "battlefield") {
			t.Fatalf("validation of %s = %v", card.Name(), err)
		}
	}
}
