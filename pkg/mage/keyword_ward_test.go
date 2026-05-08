package mage

import (
	"testing"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// scriptedPlayer is a recordingPlayer that always accepts may-pay prompts.
// (recordingPlayer's BasePlayer default declines, which is the "decline ward
// cost" branch we want for the counter test.)
type scriptedPlayer struct {
	*recordingPlayer
	mayResponse bool
}

func (sp *scriptedPlayer) ChooseMayAbility(_ string) bool { return sp.mayResponse }

func newScriptedPlayer(name string, may bool) *scriptedPlayer {
	rp := newRecordingPlayer(name)
	return &scriptedPlayer{recordingPlayer: rp, mayResponse: may}
}

// TestWard_CountersWhenOpponentDeclinesToPay verifies that an opposing
// targeted spell is countered when the opponent declines (or can't pay) the
// ward cost.
func TestWard_CountersWhenOpponentDeclinesToPay(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B") // BasePlayer.ChooseMayAbility returns false
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 20 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}

	// Warded creature on A's side.
	warded := NewCreature("Warded Bear", "{1}{G}", 2, 2, WithWard(ManaCostOf("{2}")))
	warded.SetOwner(a.PlayerID())
	wp := g.PutOnBattlefield(warded, a.PlayerID())
	g.ResolveStack()

	// B casts a hostile single-target spell (Shock-style) at the warded creature.
	shock := NewInstant("Test Shock", "{R}",
		NewTargetedSpell(TargetCreature(), DealDamage(Fixed(2))))
	shock.SetOwner(b.PlayerID())
	b.AddToHand(shock)
	b.chooseQueue = [][]uuid.UUID{{wp.ID()}}

	// Give B mana to cast.
	for range 5 {
		b.ManaPool().Add(Red, 1)
	}

	if err := g.CastSpellByID(b.PlayerID(), shock.ID(), []uuid.UUID{wp.ID()}, 0); err != nil {
		t.Fatalf("CastSpellByID: %v", err)
	}

	// Ward trigger should be queued.
	g.ResolveStack()

	// Shock should have been countered: warded creature still alive at full HP.
	if g.FindPermanent(wp.ID()) == nil {
		t.Errorf("expected warded creature to remain on the battlefield (spell countered by ward)")
	}
	if dmg := wp.Damage; dmg != 0 {
		t.Errorf("expected no damage (ward countered the spell); got Damage=%d", dmg)
	}
	// Stack should be empty.
	if !g.stack.IsEmpty() {
		t.Errorf("expected stack empty; got size %d", g.stack.Size())
	}
}

// TestWard_DoesNotTriggerOnOwnSpell verifies that ward does NOT trigger when
// the warded permanent is targeted by its own controller's spell.
func TestWard_DoesNotTriggerOnOwnSpell(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 20 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}

	warded := NewCreature("Warded Bear", "{1}{G}", 2, 2, WithWard(ManaCostOf("{2}")))
	warded.SetOwner(a.PlayerID())
	wp := g.PutOnBattlefield(warded, a.PlayerID())
	g.ResolveStack()

	// A casts a friendly (untap) on her own warded creature.
	g.TapPermanent(wp)
	untap := NewInstant("Self Untap", "{U}",
		NewTargetedSpell(TargetCreature(), UntapTarget()))
	untap.SetOwner(a.PlayerID())
	a.AddToHand(untap)
	a.chooseQueue = [][]uuid.UUID{{wp.ID()}}
	for range 5 {
		a.ManaPool().Add(Blue, 1)
	}

	if err := g.CastSpellByID(a.PlayerID(), untap.ID(), []uuid.UUID{wp.ID()}, 0); err != nil {
		t.Fatalf("CastSpellByID: %v", err)
	}
	g.ResolveStack()

	// Spell resolved, creature untapped.
	if wp.Tapped {
		t.Errorf("expected creature untapped (own ward should not trigger)")
	}
}

// TestWard_OpponentPaysAvoidCounter verifies that when the opponent accepts
// and pays the ward cost, the spell is NOT countered.
func TestWard_OpponentPaysAvoidCounter(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newScriptedPlayer("B", true) // accepts the ward may-pay prompt
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 20 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}

	warded := NewCreature("Warded Bear", "{1}{G}", 2, 2, WithWard(ManaCostOf("{2}")))
	warded.SetOwner(a.PlayerID())
	wp := g.PutOnBattlefield(warded, a.PlayerID())
	g.ResolveStack()

	shock := NewInstant("Test Shock", "{R}",
		NewTargetedSpell(TargetCreature(), DealDamage(Fixed(2))))
	shock.SetOwner(b.PlayerID())
	b.AddToHand(shock)
	b.chooseQueue = [][]uuid.UUID{{wp.ID()}}

	// B has plenty of mana to pay the spell + ward cost.
	for range 5 {
		b.ManaPool().Add(Red, 1)
	}
	for range 5 {
		b.ManaPool().Add(Blue, 1)
	}

	if err := g.CastSpellByID(b.PlayerID(), shock.ID(), []uuid.UUID{wp.ID()}, 0); err != nil {
		t.Fatalf("CastSpellByID: %v", err)
	}
	g.ResolveStack()

	// Shock should have resolved: damage dealt, creature dies (toughness 2, dmg 2 → SBA destroys).
	// Either it's gone (destroyed by SBA) or it has damage.
	perm := g.FindPermanent(wp.ID())
	if perm != nil && perm.Damage == 0 {
		t.Errorf("expected ward to be paid and shock to resolve (creature damaged or destroyed); perm.Damage=0")
	}
}

// TestWard_AttrIsSetForDisplay verifies that WithWard seeds the Ward Attr so
// AssertHasAbility / HasAttr report ward presence.
func TestWard_AttrIsSetForDisplay(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	warded := NewCreature("Warded Bear", "{1}{G}", 2, 2, WithWard(ManaCostOf("{2}")))
	warded.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(warded, a.PlayerID())
	g.ResolveStack()

	if !perm.HasAttr(Ward) {
		t.Errorf("expected HasAttr(Ward) on a creature built with WithWard")
	}
}
