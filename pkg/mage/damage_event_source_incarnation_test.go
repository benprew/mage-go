package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageEventSourceIncarnationAfterDepartureAndReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var events []GameEvent
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtDamageDealt, false, DrawCards(Fixed(0))).SetCondition(func(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events = append(events, *event)
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewCreature("Source", "", 2, 2)
	old := g.PutOnBattlefield(card, a.PlayerID())
	source := g.CaptureDamageSource(card.ID())
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.DealDamageToPlayerFromSource(b, 2, source)
	g.DealDamageToPermanentFromSource(returned, 1, source)
	if len(events) != 2 {
		t.Fatalf("events = %d", len(events))
	}
	for _, event := range events {
		if event.SourceIncarnation != old.incarnationID || event.SourceIncarnation == returned.incarnationID || event.SourceControllerID != a.PlayerID() {
			t.Fatalf("source identity = %s, controller = %s", event.SourceIncarnation, event.SourceControllerID)
		}
	}
}

func TestDamageEventSourceIncarnationAfterPreventionAndInBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var events []GameEvent
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtDamageDealt, false, DrawCards(Fixed(0))).SetCondition(func(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events = append(events, *event)
		return false
	}))), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(a, 2, g.CaptureDamageSource(second.ID())),
		PermanentDamageAssignment(first, 1, g.CaptureDamageSource(first.ID())),
	)
	if len(events) != 1 || events[0].SourceIncarnation != first.incarnationID {
		t.Fatalf("batch events = %+v", events)
	}
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(second.ID()))
	if len(events) != 2 || events[1].SourceIncarnation != second.incarnationID {
		t.Fatalf("later event = %+v", events)
	}
}

func TestDamageEventSpellSourceHasNoBattlefieldIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var eventSource uuid.UUID
	var eventIncarnation uuid.UUID
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtDamageDealt, false, DrawCards(Fixed(0))).SetCondition(func(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		eventSource, eventIncarnation = event.SourceID, event.SourceIncarnation
		return false
	}))), a.PlayerID())
	spell := NewInstant("Spell", "", nil)
	spell.SetOwner(a.PlayerID())
	end := g.resolution.Begin(&StackObject{Card: spell, Controller: b.PlayerID(), SourceID: spell.ID()})
	defer end()
	if err := ApplyEffect(g, DealDamageFrom(This, You, Fixed(1)), spell.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if eventSource != spell.ID() || eventIncarnation != uuid.Nil {
		t.Fatalf("spell source = %s, battlefield incarnation = %s", eventSource, eventIncarnation)
	}
}
