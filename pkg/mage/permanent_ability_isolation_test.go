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
	card := NewCreature("Limited", "{0}", 1, 1, WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn())))
	p := g.PutOnBattlefield(card, owner)
	action := p.RuntimeAbilities[0].(*ActionDefinition)
	action.activatedThisTurn = true
	action.activationsThisTurn = 1
	g.zones.CaptureLKI(p, g)
	snapshot := g.zones.LKI(p.ID()).Snapshot
	snapshot.RuntimeAbilities[0].SetController(uuid.New())
	if action.Controller() != owner {
		t.Error("LKI changed live ability context")
	}
	snapshot.Card.SetBasePT(8, 8)
	if card.Power() != 1 {
		t.Error("LKI changed live card")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, owner)
	if returned.RuntimeAbilities[0].(*ActionDefinition).activatedThisTurn {
		t.Error("returned permanent retained activation limit")
	}
	if card.Abilities()[0].(*ActionDefinition).activatedThisTurn {
		t.Error("runtime changed card ability")
	}
	returned.printedAbilitiesSuppressed = true
	returned.RuntimeAbilities = nil
	g.ApplyContinuousEffects()
	if returned.RuntimeAbilities[0] == card.Abilities()[0] {
		t.Error("restored ability aliases card")
	}
}

func TestNewPermanentResetsCardActivationState(t *testing.T) {
	card := NewCreature("Limited", "{0}", 1, 1, WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn(), WithTarget(TargetCreature()))))
	ability := card.abilities[0].(*ActionDefinition)
	ability.activatedThisTurn = true
	ability.activationsThisTurn = 1
	if err := ability.Targets()[0].Choose(uuid.Nil(), card, nil, []uuid.UUID{uuid.New()}); err != nil {
		t.Fatal(err)
	}
	p := NewPermanent(card, uuid.New())
	runtime := p.RuntimeAbilities[0].(*ActionDefinition)
	if runtime.activatedThisTurn || runtime.activationsThisTurn != 0 || runtime.Targets()[0].IsChosen() {
		t.Fatal("new permanent kept prior-zone ability state")
	}
}
