package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestMixedDamageBatchObservesAllRecipientsTogether(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Recipient", "", 3, 3), b.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	g.AddPreventionShield(b.PlayerID(), 1)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if a.Life() != 18 || b.Life() != 19 || creature.Damage != 2 {
			t.Errorf("callback saw partial batch: A=%d B=%d creature=%d", a.Life(), b.Life(), creature.Damage)
		}
	})
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(a, 2, captured),
		PermanentDamageAssignment(creature, 2, captured),
		PlayerDamageAssignment(b, 2, captured),
	)
	if observed != 3 {
		t.Fatalf("callbacks = %d, want 3", observed)
	}
}

func TestMixedDamageBatchKeepsSourceAndCloneState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 1, WithKeyword(Lifelink))
	source := g.PutOnBattlefield(card, a.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	clone.DealSimultaneousDamage(PlayerDamageAssignment(clone.GetPlayer(b.PlayerID()), 2, captured))
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 20 {
		t.Fatalf("original recipient life = %d, want 20", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone lifelink life = %d, want 22", got)
	}
}

func TestMixedDamageBatchDoesNotFollowReturnedOrPhasedRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Recipient", "", 2, 2)
	old := g.PutOnBattlefield(card, b.PlayerID())
	assignment := PermanentDamageAssignment(old, 2, g.CaptureDamageSource(source.ID()))
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	g.DealSimultaneousDamage(assignment)
	if current.Damage != 0 {
		t.Fatal("damage followed a returned recipient")
	}
	assignment = PermanentDamageAssignment(current, 2, g.CaptureDamageSource(source.ID()))
	g.PhaseOut(current.ID())
	g.DealSimultaneousDamage(assignment)
	if current.Damage != 0 {
		t.Fatal("phased recipient received damage")
	}
}
