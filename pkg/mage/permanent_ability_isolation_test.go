package mage

import (
	"testing"
	"uuid"
)

type wrappedClonePlayer struct{ Player }

func TestFallbackPlayerCardIsolation(t *testing.T) {
	g := newPriorityTestGame()
	card := NewCreature("Card", "{0}", 1, 1)
	g.players[0].AddToHand(card)
	g.players[0] = &wrappedClonePlayer{g.players[0]}
	g.stack.Push(&StackObject{Card: card, TargetSource: card})
	branch := g.Clone()
	cloned := branch.players[0].Hand()[0]
	if cloned == card || cloned != branch.stack.Peek().Card {
		t.Fatal("fallback did not isolate cards and preserve aliases")
	}
}

func TestPermanentAbilityLifetimeIsolation(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewArtifact("Limited", "{0}", WithAbility(NewActivated(ManaCostOf("{0}"), GainLife(1), WithOncePerTurn())))
	p := g.PutOnBattlefield(card, owner)
	if err := g.ActivateAbilityByIndex(owner, p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.zones.CaptureLKI(p, g)
	snapshot := g.zones.LKI(p.ID()).Snapshot
	snapshot.Card.SetBasePT(8, 8)
	if card.Power() != 0 {
		t.Fatal("LKI changed live card")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, owner)
	if err := g.ActivateAbilityByIndex(owner, returned.ID(), 0, nil); err != nil {
		t.Fatal("new incarnation retained count:", err)
	}

}

func TestNewPermanentResetsCardActivationState(t *testing.T) {
	card := NewArtifact("Limited", "{0}", WithAbility(NewActivated(ManaCostOf("{0}"), GainLife(1), WithOncePerTurn(), WithTarget(TargetCreature()))))
	first, second := NewPermanent(card, uuid.New()), NewPermanent(card, uuid.New())
	if first.incarnationID == second.incarnationID {
		t.Fatal("permanents share an incarnation")
	}
	if first.RuntimeAbilities[0].AbilityID() != second.RuntimeAbilities[0].AbilityID() {
		t.Fatal("definition identity changed")
	}

}
