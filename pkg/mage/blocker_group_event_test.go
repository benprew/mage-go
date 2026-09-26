package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlockerGroupEventGroupsNewLegalPairs(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 4, 4)
	first := NewCreature("First", "", 1, 2)
	second := NewCreature("Second", "", 1, 2)
	for _, card := range []Card{attacker, first, second} {
		owner := b.PlayerID()
		if card == attacker {
			owner = a.PlayerID()
		}
		g.PutOnBattlefield(card, owner)
	}
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	var events []GameEvent
	listener := NewTriggered(EvtBlockerGroup, false).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		copyEvent := *evt
		copyEvent.RelatedIDs = append([]uuid.UUID(nil), evt.RelatedIDs...)
		events = append(events, copyEvent)
		return false
	})
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(listener))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: attacker.ID()}})
	if len(events) != 3 {
		t.Fatalf("got %d grouped events, want 3", len(events))
	}
	var attackerEvent *GameEvent
	for i := range events {
		if events[i].SourceID == attacker.ID() {
			attackerEvent = &events[i]
		}
	}
	if attackerEvent == nil || attackerEvent.Flag || len(attackerEvent.RelatedIDs) != 2 || attackerEvent.RelatedIDs[0] != first.ID() || attackerEvent.RelatedIDs[1] != second.ID() {
		t.Fatalf("wrong attacker group: %+v", attackerEvent)
	}
	for _, evt := range events {
		if evt.SourceID == first.ID() || evt.SourceID == second.ID() {
			if !evt.Flag || len(evt.RelatedIDs) != 1 || evt.RelatedIDs[0] != attacker.ID() {
				t.Fatalf("wrong blocker group: %+v", evt)
			}
		}
	}
	events = nil
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}})
	if len(events) != 0 {
		t.Fatalf("old pair produced %d grouped events", len(events))
	}
}

func TestBlockerGroupEventRejectsIllegalPairsAndEntryBlocking(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 4, 4, WithStaticAbility(SourceCantBeBlockedByFewerThan(3)))
	first, second := NewCreature("First", "", 1, 2), NewCreature("Second", "", 1, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	var events []GameEvent
	g.PutOnBattlefield(NewCreature("Watcher", "", 1, 1, WithAbility(NewTriggered(EvtBlockerGroup, false).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events = append(events, *evt)
		return false
	}))), a.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: attacker.ID()}})
	if len(events) != 0 {
		t.Fatalf("illegal pairs produced %d events", len(events))
	}
	entry := g.PutOnBattlefieldBlocking(NewCreature("Entering", "", 1, 1), b.PlayerID(), attacker.ID())
	if len(events) != 1 || events[0].SourceID != attacker.ID() || events[0].Flag || len(events[0].RelatedIDs) != 1 || events[0].RelatedIDs[0] != entry.ID() {
		t.Fatalf("wrong entry group: %+v", events)
	}
}

func TestBlockerGroupEventSubtypeAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := func(evt *GameEvent, reader GameReader, source, _ uuid.UUID) bool {
		if evt.SourceID != source || evt.Flag {
			return false
		}
		for _, id := range evt.RelatedIDs {
			if permanent := reader.FindPermanent(id); permanent != nil && permanent.HasSubType("Orc") {
				return true
			}
		}
		return false
	}
	attacker := NewCreature("Attacker", "", 4, 4, WithAbility(
		NewTriggered(EvtBlockerGroup, false, GainLifeFor(You, Fixed(1))).SetCondition(condition)))
	g.PutOnBattlefield(attacker, a.PlayerID())
	first := NewCreature("Human", "", 1, 2, WithSubTypes("Human"))
	second := NewCreature("Orc 1", "", 1, 2, WithSubTypes("Orc"))
	third := NewCreature("Orc 2", "", 1, 2, WithSubTypes("Orc"))
	for _, card := range []Card{first, second, third} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	clone := g.Clone()
	clone.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: attacker.ID()}, {BlockerID: third.ID(), AttackerID: attacker.ID()}})
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("two Orcs produced life %d, want 21", got)
	}
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 20 {
		t.Fatalf("clone changed original life to %d", got)
	}
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}})
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 20 {
		t.Fatalf("non-Orc produced life %d", got)
	}
}

func TestBlockerGroupEventPendingCloneCopiesMembers(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	original := NewTriggerSystem()
	original.AddPending(&pendingTrigger{event: &GameEvent{Type: EvtBlockerGroup, RelatedIDs: []uuid.UUID{first}}})
	clone := original.Clone()
	clone.pending[0].event.RelatedIDs[0] = second
	if got := original.pending[0].event.RelatedIDs[0]; got != first {
		t.Fatalf("clone changed original event member to %s", got)
	}
}

func TestBlockerGroupEventOneSourceBlocksTwoAttackers(t *testing.T) {
	a := NewBasePlayer("A")
	b := &fixedBlockPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	blocker := NewCreature("Blocker", "", 1, 5, WithKeyword(CanBlockAny))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttacker(first.ID(), b.PlayerID())
	g.combat.AddAttacker(second.ID(), b.PlayerID())
	var matched []GameEvent
	g.PutOnBattlefield(NewCreature("Watcher", "", 1, 1, WithAbility(NewTriggered(EvtBlockerGroup, false).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == blocker.ID() {
			matched = append(matched, *evt)
		}
		return false
	}))), b.PlayerID())
	b.assignments = []BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}, {BlockerID: blocker.ID(), AttackerID: second.ID()}}
	g.doDeclareBlockers()
	if len(matched) != 1 || !matched[0].Flag || len(matched[0].RelatedIDs) != 2 || matched[0].RelatedIDs[0] != first.ID() || matched[0].RelatedIDs[1] != second.ID() {
		t.Fatalf("wrong blocking group: %+v", matched)
	}
}
