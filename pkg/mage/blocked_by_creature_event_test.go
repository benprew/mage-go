package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type blockedByTestCondition struct{ blocker uuid.UUID }

func (c blockedByTestCondition) CheckTriggerCond(event *GameEvent, _ GameReader, source, _ uuid.UUID) bool {
	return event.SourceID == source && event.TargetID == c.blocker
}

func TestBlockedByCreatureEventPairsAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 1, 1)
	attacker := NewCreature("Attacker", "", 4, 4, WithAbility(
		NewTriggered(EvtCreatureBlockedBy, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{}),
	))
	attacker.SetOwner(a.PlayerID())
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	clone := g.Clone()
	clone.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: attacker.ID()}})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("two blocker pairs gave life %d, want 22", got)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("clone changed original life: %d", got)
	}
	clone.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("old pair fired again: %d", got)
	}
}

func TestBlockedByCreatureEventEntryAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 1, 1)
	attacker := NewCreature("Attacker", "", 4, 4, WithAbility(
		NewTriggered(EvtCreatureBlockedBy, false, GainLifeFor(You, Fixed(1))).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{EventSourceIsSelf{}, blockedByTestCondition{blocker: first.ID()}}}),
	))
	attacker.SetOwner(a.PlayerID())
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}})
	g.PutOnBattlefieldBlocking(NewCreature("Entering", "", 1, 1), b.PlayerID(), attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("wrong blocker identity or entry duplicate: life %d", got)
	}
}

func TestBlockedByCreatureEventRejectsIllegalDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 4, 4,
		WithAbility(NewTriggered(EvtCreatureBlockedBy, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{})),
		WithStaticAbility(SourceCantBeBlockedByFewerThan(3)))
	attacker.SetOwner(a.PlayerID())
	first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: attacker.ID()}})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("illegal blockers triggered: life %d", got)
	}
}

func TestBlockedByCreatureEventKeepsTriggerController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 4, 4, WithAbility(
		NewTriggered(EvtCreatureBlockedBy, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{}),
	))
	attacker.SetOwner(a.PlayerID())
	blocker := NewCreature("Blocker", "", 1, 1)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("trigger used returned source controller: A %d, B %d", a.Life(), b.Life())
	}
}

func TestBlockedByCreatureEventNoAttacker(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 4, 4, WithAbility(
		NewTriggered(EvtCreatureBlockedBy, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{}),
	))
	attacker.SetOwner(a.PlayerID())
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefieldBlocking(NewCreature("Entering", "", 1, 1), b.PlayerID(), attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("entry without an attacking creature triggered: life %d", got)
	}
}
