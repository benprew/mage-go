package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestDamageHistoryPermanentIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 1, 3)
	recipientCard := NewCreature("Recipient", "", 1, 4)
	oldSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	oldRecipient := g.PutOnBattlefield(recipientCard, b.PlayerID())
	source := g.CaptureDamageSource(oldSource.ID())
	g.DealDamageToPlayerFromSource(b, 1, source)
	g.DealDamageToPermanentFromSource(oldRecipient, 1, source)
	if !g.HasDealtDamageToPlayerFromSource(source, b.PlayerID()) || !g.HasDealtDamageToPermanentFromSource(source, oldRecipient) {
		t.Fatal("positive damage did not enter incarnation history")
	}
	g.RemoveFromBattlefield(oldSource)
	g.RemoveFromBattlefield(oldRecipient)
	newSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	newRecipient := g.PutOnBattlefield(recipientCard, b.PlayerID())
	g.zones.ClearLKI()
	if g.HasDealtDamageToPlayerFromSource(g.CaptureDamageSource(newSource.ID()), b.PlayerID()) || g.HasDealtDamageToPermanentFromSource(source, newRecipient) {
		t.Fatal("returned object inherited damage history")
	}
	if !g.HasDealtDamageToPlayerFromSource(source, b.PlayerID()) || !g.HasDealtDamageToPermanentFromSource(source, oldRecipient) {
		t.Fatal("departure erased old damage history")
	}
	clone := g.Clone()
	cloneSource := clone.CaptureDamageSource(newSource.ID())
	clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, cloneSource)
	if !clone.HasDealtDamageToPlayerFromSource(cloneSource, b.PlayerID()) || g.HasDealtDamageToPlayerFromSource(cloneSource, b.PlayerID()) {
		t.Fatal("cloned history shares new source writes")
	}
	g.DealDamageToPermanentFromSource(newRecipient, 1, source)
	if !g.HasDealtDamageToPermanentFromSource(source, newRecipient) || g.HasDealtDamageToPermanentFromSource(g.CaptureDamageSource(newSource.ID()), newRecipient) {
		t.Fatal("damage from departed source used returned source identity")
	}
}

func TestDamageHistoryOutcomesPhasingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
	recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 4), b.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	g.AddPreventionShield(recipient.ID(), 2)
	g.DealDamageToPermanentFromSource(recipient, 2, captured)
	if g.HasDealtDamageToPermanentFromSource(captured, recipient) {
		t.Fatal("prevented damage entered history")
	}
	g.PhaseOut(recipient.ID())
	g.DealDamageToPermanentFromSource(recipient, 1, captured)
	if g.HasDealtDamageToPermanentFromSource(captured, recipient) {
		t.Fatal("phased-out recipient received damage")
	}
	g.PhaseIn([]uuid.UUID{recipient.ID()})
	clone := g.Clone()
	cloneRecipient := clone.FindPermanent(recipient.ID())
	clone.DealSimultaneousDamage(PermanentDamageAssignment(cloneRecipient, 1, captured), PlayerDamageAssignment(clone.GetPlayer(b.PlayerID()), 1, captured))
	if !clone.HasDealtDamageToPermanentFromSource(captured, cloneRecipient) || !clone.HasDealtDamageToPlayerFromSource(captured, b.PlayerID()) {
		t.Fatal("simultaneous damage did not enter clone history")
	}
	if g.HasDealtDamageToPermanentFromSource(captured, recipient) || g.HasDealtDamageToPlayerFromSource(captured, b.PlayerID()) {
		t.Fatal("clone changed original history")
	}
	g.DealDamageToPlayerFromSource(b, 1, captured)
	if clone.HasDealtDamageToPlayerFromSource(DamageSource{}, a.PlayerID()) {
		t.Fatal("absent source has history")
	}
	if !g.HasDealtDamageToPlayerFromSource(captured, b.PlayerID()) {
		t.Fatal("original damage did not enter history")
	}
}
