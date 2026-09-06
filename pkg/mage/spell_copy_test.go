package mage

import (
	"testing"

	"github.com/google/uuid"
)

// CR 706 — Copying spells.
//
// Game.CopySpellOnStack creates a duplicate StackObject that shares the
// original spell's effects and X value, has its own object ID, is marked
// IsCopy (so it ceases to exist on resolve / counter, CR 707.10), and
// either inherits the original's targets or re-prompts the controller for
// new ones (CR 706.10c).

func newSpellCopyGame() (*Game, *recordingPlayer, *recordingPlayer) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 30 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}
	return g, a, b
}

// TestCopySpellOnStack_SameTargets: copy inherits the original target. The
// original spell and the copy each apply their effect to that target.
func TestCopySpellOnStack_SameTargets(t *testing.T) {
	g, a, b := newSpellCopyGame()

	bolt := NewInstant("Bolt", "{R}",
		NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(3))),
	)
	bolt.SetOwner(a.PlayerID())
	a.AddToHand(bolt)

	// Manually push the original onto the stack with PlayerB as target.
	orig := &StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: a.PlayerID(),
		SourceID:   bolt.ID(),
		Effects:    []Effect{DealDamage(Fixed(3))},
		Targets:    []uuid.UUID{b.PlayerID()},
	}
	a.RemoveFromHand(bolt.ID())
	g.pushStack(orig)

	// Copy the spell, inheriting targets.
	cp := g.CopySpellOnStack(bolt.ID(), a.PlayerID(), false)
	if cp == nil {
		t.Fatalf("CopySpellOnStack returned nil")
	}
	if !cp.IsCopy {
		t.Errorf("copy is not marked IsCopy")
	}
	if cp.ID == orig.ID {
		t.Errorf("copy must have a fresh ID")
	}
	if cp.Card == nil || cp.Card.ID() == bolt.ID() {
		t.Errorf("copy must have a fresh Card with its own ID")
	}
	if len(cp.Targets) != 1 || cp.Targets[0] != b.PlayerID() {
		t.Errorf("copy targets: got %v, want %v", cp.Targets, []uuid.UUID{b.PlayerID()})
	}

	g.ResolveStack()

	// PlayerB took 3 from copy + 3 from original = 6.
	if b.Life() != 14 {
		t.Errorf("PlayerB life: got %d, want 14", b.Life())
	}
}

// TestCopySpellOnStack_NewTargets: with mayChooseNewTargets=true, the
// controller of the copy is reprompted via ChooseTargets and may pick a
// different legal target.
func TestCopySpellOnStack_NewTargets(t *testing.T) {
	g, a, b := newSpellCopyGame()

	bolt := NewInstant("Bolt", "{R}",
		NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(3))),
	)
	bolt.SetOwner(a.PlayerID())
	a.AddToHand(bolt)

	orig := &StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: a.PlayerID(),
		SourceID:   bolt.ID(),
		Effects:    []Effect{DealDamage(Fixed(3))},
		Targets:    []uuid.UUID{b.PlayerID()},
	}
	a.RemoveFromHand(bolt.ID())
	g.pushStack(orig)

	// Script: when reprompted, A picks itself.
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}

	cp := g.CopySpellOnStack(bolt.ID(), a.PlayerID(), true)
	if cp == nil {
		t.Fatalf("CopySpellOnStack returned nil")
	}
	if len(cp.Targets) != 1 || cp.Targets[0] != a.PlayerID() {
		t.Errorf("copy targets after reprompt: got %v, want PlayerA", cp.Targets)
	}

	g.ResolveStack()

	// PlayerA took 3 (from copy with new target); PlayerB took 3 (original).
	if a.Life() != 17 {
		t.Errorf("PlayerA life: got %d, want 17", a.Life())
	}
	if b.Life() != 17 {
		t.Errorf("PlayerB life: got %d, want 17", b.Life())
	}
}

// TestCopySpellOnStack_CopyDoesNotEnterAnyZone: per CR 707.10, a copy of a
// spell ceases to exist as it resolves. The original Card goes to graveyard
// (ResolveStackObject's normal path); the copy's card does NOT.
func TestCopySpellOnStack_CopyDoesNotEnterAnyZone(t *testing.T) {
	g, a, b := newSpellCopyGame()

	bolt := NewInstant("Bolt", "{R}",
		NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(3))),
	)
	bolt.SetOwner(a.PlayerID())
	a.AddToHand(bolt)

	orig := &StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: a.PlayerID(),
		SourceID:   bolt.ID(),
		Effects:    []Effect{DealDamage(Fixed(3))},
		Targets:    []uuid.UUID{b.PlayerID()},
	}
	a.RemoveFromHand(bolt.ID())
	g.pushStack(orig)

	cp := g.CopySpellOnStack(bolt.ID(), a.PlayerID(), false)
	copyCardID := cp.Card.ID()
	g.ResolveStack()

	// Original Bolt is in PlayerA's graveyard.
	gy := a.Graveyard()
	if len(gy) != 1 {
		t.Fatalf("PlayerA graveyard: got %d cards, want 1", len(gy))
	}
	if gy[0].ID() != bolt.ID() {
		t.Errorf("graveyard card: got %v, want original bolt %v", gy[0].ID(), bolt.ID())
	}
	// The copy's card must not appear anywhere — not graveyard, not exile,
	// not battlefield, not hand, not library.
	if cardSliceContains(gy, copyCardID) {
		t.Errorf("copy card found in graveyard; should have ceased to exist")
	}
	if cardSliceContains(a.Hand(), copyCardID) || cardSliceContains(a.Library(), copyCardID) {
		t.Errorf("copy card found in hand/library; should have ceased to exist")
	}
	for _, ec := range g.zones.exile {
		if ec.Card.ID() == copyCardID {
			t.Errorf("copy card found in exile; should have ceased to exist")
		}
	}
	for _, p := range g.zones.battlefield {
		if p.Card.ID() == copyCardID {
			t.Errorf("copy card found on battlefield; should have ceased to exist")
		}
	}
}

func cardSliceContains(cards []Card, id uuid.UUID) bool {
	for _, c := range cards {
		if c.ID() == id {
			return true
		}
	}
	return false
}

// TestCopySpellOnStack_NotFoundReturnsNil: if the source spell isn't on the
// stack, CopySpellOnStack returns nil (caller can safely no-op).
func TestCopySpellOnStack_NotFoundReturnsNil(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	cp := g.CopySpellOnStack(uuid.New(), a.PlayerID(), false)
	if cp != nil {
		t.Errorf("CopySpellOnStack for missing source: got %v, want nil", cp)
	}
}

// TestCopySpellOnStack_FizzledCopyCeases: a copy whose targets all become
// illegal still ceases to exist (no graveyard / battlefield) when it
// fizzles, per CR 707.10.
func TestCopySpellOnStack_FizzledCopyCeases(t *testing.T) {
	g, a, b := newSpellCopyGame()

	// Build a creature that the spell can target.
	target := NewCreature("Target Creature", "{1}", 1, 1)
	target.SetOwner(b.PlayerID())
	perm := g.PutOnBattlefield(target, b.PlayerID())

	dmg := NewInstant("Dmg", "{R}",
		NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(3))),
	)
	dmg.SetOwner(a.PlayerID())
	a.AddToHand(dmg)

	orig := &StackObject{
		ID:         uuid.New(),
		Card:       dmg,
		Controller: a.PlayerID(),
		SourceID:   dmg.ID(),
		Effects:    []Effect{DealDamage(Fixed(3))},
		Targets:    []uuid.UUID{perm.ID()},
	}
	a.RemoveFromHand(dmg.ID())
	g.pushStack(orig)

	cp := g.CopySpellOnStack(dmg.ID(), a.PlayerID(), false)
	copyCardID := cp.Card.ID()

	// Remove the target before resolution → both fizzle.
	g.RemoveFromBattlefield(perm)

	g.ResolveStack()

	// Copy ceases without going to graveyard.
	for _, c := range a.Graveyard() {
		if c.ID() == copyCardID {
			t.Errorf("fizzled copy went to graveyard; should have ceased to exist")
		}
	}
	// Original (a real card) goes to its owner's graveyard.
	if !cardSliceContains(a.Graveyard(), dmg.ID()) {
		t.Errorf("original fizzled spell should be in graveyard")
	}
}
