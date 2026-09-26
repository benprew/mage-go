package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageEventKeepsDepartedSourceController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var controllers []uuid.UUID
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtDamageDealt, false, DrawCards(Fixed(0))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		controllers = append(controllers, evt.SourceControllerID)
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	source := NewCreature("Source", "", 2, 2)
	g.PutOnBattlefield(source, a.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.DealDamageToPlayerFromSource(b, 2, captured)
	if len(controllers) != 1 || controllers[0] != a.PlayerID() {
		t.Fatalf("damage source controllers = %v", controllers)
	}
}

func TestDamageEventControllerAfterPreventionAndInBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var controllers []uuid.UUID
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtDamageDealt, false, DrawCards(Fixed(0))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		controllers = append(controllers, evt.SourceControllerID)
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 1)
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(a, 2, g.CaptureDamageSource(second.ID())),
		PermanentDamageAssignment(g.FindPermanent(first.ID()), 1, g.CaptureDamageSource(first.ID())),
	)
	if len(controllers) != 2 || controllers[0] != b.PlayerID() || controllers[1] != a.PlayerID() {
		t.Fatalf("batch source controllers = %v", controllers)
	}
}

func TestDamageEventSpellControllerAndFullPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var controllers []uuid.UUID
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtDamageDealt, false, DrawCards(Fixed(0))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		controllers = append(controllers, evt.SourceControllerID)
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "", nil)
	spell.SetOwner(a.PlayerID())
	end := g.resolution.Begin(&StackObject{Card: spell, Controller: b.PlayerID(), SourceID: spell.ID()})
	defer end()
	g.AddPreventionShield(b.PlayerID(), 2)
	if err := ApplyEffect(g, DealDamageFrom(This, You, Fixed(2)), spell.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(controllers) != 0 {
		t.Fatalf("prevented damage events = %v", controllers)
	}
	if err := ApplyEffect(g, DealDamageFrom(This, You, Fixed(1)), spell.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(controllers) != 1 || controllers[0] != b.PlayerID() {
		t.Fatalf("spell source controllers = %v", controllers)
	}
	if g.StackSize() != 0 {
		t.Fatal("observation trigger unexpectedly entered the stack")
	}
}
