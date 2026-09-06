package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TestStunCounter_RemovedInsteadOfUntapping verifies CR 122.1g: a tapped
// permanent with one stun counter is not untapped during the untap step;
// instead the stun counter is removed.
func TestStunCounter_RemovedInsteadOfUntapping(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 20 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}

	bear := NewCreature("Stunned Bear", "{1}{G}", 2, 2)
	bear.SetOwner(a.PlayerID())
	bp := g.PutOnBattlefield(bear, a.PlayerID())
	g.ResolveStack()

	g.TapPermanent(bp)
	bp.AddCounter(Stun, 2)

	g.SetActivePlayerIndex(0) // index of player A
	g.doUntap()

	if !bp.Tapped {
		t.Errorf("expected stunned permanent to remain tapped after untap step")
	}
	if bp.Counters[Stun] != 1 {
		t.Errorf("expected exactly one stun counter to be removed; got %d remaining", bp.Counters[Stun])
	}
}

// TestStunCounter_UntapsWhenNoCountersRemain verifies that after the last
// stun counter is consumed (in a previous untap step), a subsequent untap
// step actually untaps the permanent.
func TestStunCounter_UntapsWhenNoCountersRemain(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	bear := NewCreature("Stunned Bear", "{1}{G}", 2, 2)
	bear.SetOwner(a.PlayerID())
	bp := g.PutOnBattlefield(bear, a.PlayerID())
	g.ResolveStack()

	g.TapPermanent(bp)
	bp.AddCounter(Stun, 1)
	g.SetActivePlayerIndex(0) // index of player A

	g.doUntap()
	if !bp.Tapped {
		t.Fatalf("first untap must consume stun counter, leaving permanent tapped")
	}
	if bp.Counters[Stun] != 0 {
		t.Fatalf("expected 0 stun counters; got %d", bp.Counters[Stun])
	}

	g.doUntap()
	if bp.Tapped {
		t.Errorf("expected permanent to untap on second untap step (no stun counters left)")
	}
}

// TestStunCounter_RemovedByEffectDrivenUntap verifies that effect-driven
// untaps (UntapPermanent) also consume stun counters instead of untapping.
func TestStunCounter_RemovedByEffectDrivenUntap(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	bear := NewCreature("Stunned Bear", "{1}{G}", 2, 2)
	bear.SetOwner(a.PlayerID())
	bp := g.PutOnBattlefield(bear, a.PlayerID())
	g.ResolveStack()

	g.TapPermanent(bp)
	bp.AddCounter(Stun, 1)

	if g.UntapPermanent(bp) {
		t.Errorf("UntapPermanent should not untap a permanent with a stun counter")
	}
	if !bp.Tapped {
		t.Errorf("expected permanent to remain tapped")
	}
	if bp.Counters[Stun] != 0 {
		t.Errorf("expected stun counter consumed; got %d", bp.Counters[Stun])
	}
}
