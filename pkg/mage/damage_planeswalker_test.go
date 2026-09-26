package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlaneswalkerDamageRemovesLoyaltyAfterPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	walker := g.PutOnBattlefield(NewPlaneswalker("Test Walker", "", 5), b.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Deathtouch)), a.PlayerID())
	if got := walker.Counters[Loyalty]; got != 5 {
		t.Fatalf("starting loyalty = %d, want 5", got)
	}
	g.AddPreventionShield(walker.ID(), 2)
	g.DealDamageToPermanent(walker, 3, source.ID())
	if got := walker.Counters[Loyalty]; got != 4 {
		t.Errorf("loyalty after one damage = %d, want 4", got)
	}
	if walker.Damage != 0 {
		t.Errorf("damage marked on planeswalker = %d, want 0", walker.Damage)
	}
	if !g.HasDealtDamageToPermanent(source.ID(), walker.ID()) {
		t.Error("actual damage was not recorded")
	}
	g.DealDamageToPermanent(walker, 1, source.ID())
	if got := walker.Counters[Loyalty]; got != 3 {
		t.Errorf("loyalty after shield is spent = %d, want 3", got)
	}
}

func TestPlaneswalkerFullyPreventedDamageHasNoOutcome(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	walker := g.PutOnBattlefield(NewPlaneswalker("Test Walker", "", 3), b.PlayerID())
	g.AddPreventionShield(walker.ID(), 2)
	g.DealDamageToPermanent(walker, 2, uuid.Nil)
	if walker.Counters[Loyalty] != 3 || walker.Damage != 0 || len(g.GetDamageSources(walker.ID())) != 0 {
		t.Fatal("fully prevented damage changed planeswalker state")
	}
}

func TestPlaneswalkerDamageCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	walker := g.PutOnBattlefield(NewPlaneswalker("Test Walker", "", 4), b.PlayerID())
	clone := g.Clone()
	clone.DealDamageToPermanent(clone.FindPermanent(walker.ID()), 2, uuid.Nil)
	if got := clone.FindPermanent(walker.ID()).Counters[Loyalty]; got != 2 {
		t.Errorf("clone loyalty = %d, want 2", got)
	}
	if got := walker.Counters[Loyalty]; got != 4 {
		t.Errorf("original loyalty = %d, want 4", got)
	}
}

func TestPlaneswalkerCreatureDamageHasBothResults(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Animated Walker", "", 3, 3)
	card.AddType(TypePlaneswalker)
	walker := g.PutOnBattlefield(card, b.PlayerID())
	walker.AddCounter(Loyalty, 2)
	g.DealDamageToPermanent(walker, 3, uuid.Nil)
	if walker.Counters[Loyalty] != 0 || walker.Damage != 3 {
		t.Fatalf("loyalty = %d, marked damage = %d; want 0 and 3", walker.Counters[Loyalty], walker.Damage)
	}
}

func TestPlaneswalkerZeroLoyaltyStateAction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	walker := g.PutOnBattlefield(NewPlaneswalker("Test Walker", "", 2), b.PlayerID())
	g.DealDamageToPermanent(walker, 2, uuid.Nil)
	if g.FindPermanent(walker.ID()) == nil {
		t.Fatal("damage moved the planeswalker before state-based actions")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(walker.ID()) != nil {
		t.Fatal("zero-loyalty planeswalker remained on the battlefield")
	}
}
