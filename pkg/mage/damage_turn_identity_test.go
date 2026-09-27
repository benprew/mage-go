package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTurnDamageHistoryKeepsBothIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 1, 4)
	recipientCard := NewCreature("Recipient", "", 1, 5)
	oldSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	oldRecipient := g.PutOnBattlefield(recipientCard, b.PlayerID())
	firstSource := g.CaptureDamageSource(oldSource.ID())
	firstRecipient := capturedPermanent{oldRecipient.ID(), oldRecipient.incarnationID}
	g.DealDamageToPermanentFromSource(oldRecipient, 1, firstSource)
	if !g.damage.dealtDamageToPermanentThisTurn(firstSource, firstRecipient) {
		t.Fatal("positive damage was not recorded for both incarnations")
	}
	g.RemoveFromBattlefield(oldSource)
	g.RemoveFromBattlefield(oldRecipient)
	newSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	newRecipient := g.PutOnBattlefield(recipientCard, b.PlayerID())
	g.zones.ClearLKI()
	secondSource := g.CaptureDamageSource(newSource.ID())
	secondRecipient := capturedPermanent{newRecipient.ID(), newRecipient.incarnationID}
	if g.damage.dealtDamageToPermanentThisTurn(secondSource, secondRecipient) || g.damage.dealtDamageToPermanentThisTurn(firstSource, secondRecipient) || g.damage.dealtDamageToPermanentThisTurn(secondSource, firstRecipient) {
		t.Fatal("new incarnation inherited earlier damage")
	}
	if !g.damage.dealtDamageToPermanentThisTurn(firstSource, firstRecipient) {
		t.Fatal("departure or LKI cleanup erased first incarnation history")
	}
	clone := g.Clone()
	clone.DealDamageToPermanentFromSource(clone.FindPermanent(newRecipient.ID()), 1, secondSource)
	if !clone.damage.dealtDamageToPermanentThisTurn(secondSource, secondRecipient) || g.damage.dealtDamageToPermanentThisTurn(secondSource, secondRecipient) {
		t.Fatal("clone shared turn damage history")
	}
	g.damage.ClearEndOfTurn()
	if g.damage.dealtDamageToPermanentThisTurn(firstSource, firstRecipient) || !clone.damage.dealtDamageToPermanentThisTurn(firstSource, firstRecipient) {
		t.Fatal("turn cleanup did not clear history independently")
	}
}

func TestTurnDamageHistoryIgnoresPreventedAndPhasedDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 4), a.PlayerID())
	recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 5), b.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	key := capturedPermanent{recipient.ID(), recipient.incarnationID}
	g.AddPreventionShield(recipient.ID(), 2)
	g.DealDamageToPermanentFromSource(recipient, 2, captured)
	if g.damage.dealtDamageToPermanentThisTurn(captured, key) {
		t.Fatal("prevented damage entered turn history")
	}
	phased := g.PhaseOut(recipient.ID())
	g.DealDamageToPermanentFromSource(recipient, 1, captured)
	if g.damage.dealtDamageToPermanentThisTurn(captured, key) {
		t.Fatal("phased recipient entered turn history")
	}
	g.PhaseIn(phased)
	g.DealDamageToPermanentFromSource(recipient, 1, captured)
	if !g.damage.dealtDamageToPermanentThisTurn(captured, key) {
		t.Fatal("phasing erased recipient identity")
	}
}

func TestTurnDamageHistoryDepartedSourceAndCleanupPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 1, 4)
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 5), b.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	key := capturedPermanent{recipient.ID(), recipient.incarnationID}
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.zones.ClearLKI()
	g.DealDamageToPermanentFromSource(recipient, 1, captured)
	if !g.damage.dealtDamageToPermanentThisTurn(captured, key) || g.damage.dealtDamageToPermanentThisTurn(g.CaptureDamageSource(source.ID()), key) {
		t.Fatal("departed source damage used the returned incarnation")
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	if !g.doCleanupActions() || !g.damage.dealtDamageToPermanentThisTurn(captured, key) {
		t.Fatal("cleanup priority removed damage history early")
	}
	g.ResolveStack()
	g.RunStepWithPriority(Cleanup)
	if g.damage.dealtDamageToPermanentThisTurn(captured, key) {
		t.Fatal("final cleanup retained turn damage history")
	}
}
