package mage

import (
	"testing"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestLookupObject_LiveAndSnapshot verifies LKIView's transparent fallback:
// while a permanent is on the battlefield LookupObject returns a live view;
// after RemoveFromBattlefield captures LKI it returns the snapshot view,
// and both paths answer the same predicates (HasType, HasKeyword, Power)
// per CR 603.6c.
func TestLookupObject_LiveAndSnapshot(t *testing.T) {
	pA := NewBasePlayer("A")
	pB := NewBasePlayer("B")
	g := NewGame(pA, pB)

	card := NewCreature("LKI Test Bear", "{1}{G}", 2, 2,
		WithSubTypes("Bear"),
		WithKeyword(Trample),
	)
	card.SetOwner(pA.PlayerID())
	perm := g.PutOnBattlefield(card, pA.PlayerID())

	// Live: LookupObject hits the live permanent.
	live := g.LookupObject(perm.ID())
	if live == nil {
		t.Fatal("expected live LKIView, got nil")
	}
	if !live.ViewHasType(TypeCreature) {
		t.Errorf("live view: expected HasType(Creature) true")
	}
	if !live.ViewHasSubType("Bear") {
		t.Errorf("live view: expected HasSubType(Bear) true")
	}
	if !live.ViewHasKeyword(Trample) {
		t.Errorf("live view: expected HasKeyword(Trample) true")
	}
	if got := live.ViewPower(); got != 2 {
		t.Errorf("live view: power = %d, want 2", got)
	}
	if got := live.ViewController(); got != pA.PlayerID() {
		t.Errorf("live view: controller = %v, want %v", got, pA.PlayerID())
	}

	// Snapshot: after destruction LookupObject returns the LKI view, which
	// answers the same predicates.
	g.DestroyPermanent(perm)
	snap := g.LookupObject(perm.ID())
	if snap == nil {
		t.Fatal("expected snapshot LKIView after destroy, got nil")
	}
	if !snap.ViewHasType(TypeCreature) {
		t.Errorf("snapshot view: expected HasType(Creature) true")
	}
	if !snap.ViewHasSubType("Bear") {
		t.Errorf("snapshot view: expected HasSubType(Bear) true")
	}
	if !snap.ViewHasKeyword(Trample) {
		t.Errorf("snapshot view: expected HasKeyword(Trample) true")
	}
	if got := snap.ViewPower(); got != 2 {
		t.Errorf("snapshot view: power = %d, want 2", got)
	}
	if got := snap.ViewController(); got != pA.PlayerID() {
		t.Errorf("snapshot view: controller = %v, want %v", got, pA.PlayerID())
	}
}

// TestLookupObject_SnapshotPreservesCounters verifies the deep-clone
// snapshot retains counters and runtime abilities (Step B). A trigger
// resolving against a dead permanent reads the same counter count it had
// just before the move per CR 603.6c.
func TestLookupObject_SnapshotPreservesCounters(t *testing.T) {
	pA := NewBasePlayer("A")
	pB := NewBasePlayer("B")
	g := NewGame(pA, pB)

	card := NewCreature("LKI Counter Bear", "{1}{G}", 2, 2,
		WithSubTypes("Bear"),
	)
	card.SetOwner(pA.PlayerID())
	perm := g.PutOnBattlefield(card, pA.PlayerID())
	perm.AddCounter(P1P1, 3)

	// Live view sees 3 counters and the ETB's runtime abilities.
	live := g.LookupObject(perm.ID())
	if got := live.ViewCounters(P1P1); got != 3 {
		t.Errorf("live ViewCounters(P1P1) = %d, want 3", got)
	}

	g.DestroyPermanent(perm)

	snap := g.LookupObject(perm.ID())
	if snap == nil {
		t.Fatal("expected snapshot view, got nil")
	}
	if got := snap.ViewCounters(P1P1); got != 3 {
		t.Errorf("snapshot ViewCounters(P1P1) = %d, want 3", got)
	}
	// Mutating the live (gone) permanent's counters must not show through
	// to the snapshot - it's a deep copy.
	perm.AddCounter(P1P1, 10)
	if got := snap.ViewCounters(P1P1); got != 3 {
		t.Errorf("snapshot ViewCounters after live mutation = %d, want 3 (snapshot must be isolated)", got)
	}
}

// TestLookupObject_UnknownID returns nil rather than a misleading view.
func TestLookupObject_UnknownID(t *testing.T) {
	pA := NewBasePlayer("A")
	pB := NewBasePlayer("B")
	g := NewGame(pA, pB)
	if v := g.LookupObject(uuid.New()); v != nil {
		t.Errorf("expected nil view for unknown id, got %v", v)
	}
}
