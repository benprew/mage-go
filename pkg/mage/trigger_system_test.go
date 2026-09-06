package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTriggerSystem_PendingTriggers(t *testing.T) {
	ts := NewTriggerSystem()
	p1 := uuid.New()
	s1 := uuid.New()

	pt := &pendingTrigger{
		sourceID:   s1,
		controller: p1,
	}
	ts.AddPending(pt)

	if len(ts.Pending()) != 1 {
		t.Fatalf("expected 1 pending trigger, got %d", len(ts.Pending()))
	}
	if ts.Pending()[0].sourceID != s1 {
		t.Errorf("expected sourceID %s, got %s", s1, ts.Pending()[0].sourceID)
	}

	ts.ClearPending()
	if len(ts.Pending()) != 0 {
		t.Fatalf("expected 0 pending triggers after clear, got %d", len(ts.Pending()))
	}
}

func TestTriggerSystem_DelayedTriggers(t *testing.T) {
	ts := NewTriggerSystem()
	dt1 := &DelayedTrigger{
		EventType:  EvtEndStep,
		Persistent: false,
	}
	dt2 := &DelayedTrigger{
		EventType:  EvtEndStep,
		Persistent: true,
	}

	ts.RegisterDelayed(dt1)
	ts.RegisterDelayed(dt2)

	if len(ts.Delayed()) != 2 {
		t.Fatalf("expected 2 delayed triggers, got %d", len(ts.Delayed()))
	}

	ts.ClearEndOfTurn()
	if len(ts.Delayed()) != 1 {
		t.Fatalf("expected 1 non-persistent delayed trigger after end of turn, got %d", len(ts.Delayed()))
	}
	if ts.Delayed()[0].Persistent {
		t.Errorf("expected remaining trigger to be non-persistent (dt1)")
	}
}

func TestTriggerSystem_StateTriggers(t *testing.T) {
	ts := NewTriggerSystem()
	key1 := stateTriggerKey{sourceID: uuid.New(), abilityID: uuid.New()}
	key2 := stateTriggerKey{sourceID: uuid.New(), abilityID: uuid.New()}

	ts.ArmStateTrigger(key1)
	if !ts.IsStateTriggerArmed(key1) {
		t.Errorf("expected key1 to be armed")
	}
	if ts.IsStateTriggerArmed(key2) {
		t.Errorf("expected key2 to not be armed")
	}

	ts.DisarmStateTrigger(key1)
	if ts.IsStateTriggerArmed(key1) {
		t.Errorf("expected key1 to be disarmed")
	}

	ts.ArmStateTrigger(key1)
	ts.ArmStateTrigger(key2)
	seen := map[stateTriggerKey]bool{key1: true}
	ts.ClearUnseenArmed(seen)
	if !ts.IsStateTriggerArmed(key1) {
		t.Errorf("expected key1 to remain armed")
	}
	if ts.IsStateTriggerArmed(key2) {
		t.Errorf("expected key2 to be cleared")
	}
}

func TestTriggerSystem_CloneIsolation(t *testing.T) {
	ts := NewTriggerSystem()
	dt := &DelayedTrigger{
		EventType:  EvtCleanup,
		Persistent: true,
	}
	ts.RegisterDelayed(dt)

	key := stateTriggerKey{sourceID: uuid.New(), abilityID: uuid.New()}
	ts.ArmStateTrigger(key)

	pt := &pendingTrigger{sourceID: uuid.New()}
	ts.AddPending(pt)

	clone := ts.Clone()

	// Mutate clone
	clone.Delayed()[0].Persistent = false
	clone.DisarmStateTrigger(key)
	clone.ClearPending()

	// Verify original is untouched
	if !ts.Delayed()[0].Persistent {
		t.Errorf("original delayed trigger modified")
	}
	if !ts.IsStateTriggerArmed(key) {
		t.Errorf("original state trigger disarmed")
	}
	if len(ts.Pending()) != 1 {
		t.Errorf("original pending triggers cleared")
	}
}
