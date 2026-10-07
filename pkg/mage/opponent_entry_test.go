package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type opponentEntryProbe struct {
	BaseAbility
	check func(*Game, *Permanent)
}

func (a *opponentEntryProbe) OnEnter(g *Game, p *Permanent) bool {
	a.check(g, p)
	return true
}

func TestChooseOpponentAsEntersTiming(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	checked := 0
	probe := &opponentEntryProbe{check: func(g *Game, p *Permanent) {
		checked++
		if p.ChosenPlayer != b.PlayerID() {
			t.Fatal("opponent was not chosen before the next entry operation")
		}
		for _, live := range g.AllBattlefield() {
			if live.ID() == p.ID() {
				t.Fatal("choice ran after battlefield registration")
			}
		}
	}}
	card := NewArtifact("Entry", "", WithAbility(ChooseOpponentAsEnters()), WithAbility(probe))
	card.SetOwner(b.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if checked != 1 || g.StackSize() != 0 {
		t.Fatal("entry operation did not run exactly once without the stack")
	}
}

func TestChooseOpponentAsEntersIdentity(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	card := NewArtifact("Entry", "", WithAbility(ChooseOpponentAsEnters()))
	card.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(card, a.PlayerID())
	secondCard := card.Copy()
	secondCard.SetID(uuid.New())
	second := g.PutOnBattlefield(secondCard, b.PlayerID())
	if first.ChosenPlayer != b.PlayerID() || second.ChosenPlayer != a.PlayerID() {
		t.Fatal("instances shared an entry choice or used the owner")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	if g.FindPermanent(first.ID()).ChosenPlayer != b.PlayerID() {
		t.Fatal("control change repeated the choice")
	}
	g.PhaseIn(g.PhaseOut(first.ID()))
	if g.FindPermanent(first.ID()).ChosenPlayer != b.PlayerID() {
		t.Fatal("phase cycle repeated entry")
	}
	clone := g.Clone()
	clone.MutablePermanent(first.ID()).ChosenPlayer = a.PlayerID()
	if g.FindPermanent(first.ID()).ChosenPlayer != b.PlayerID() {
		t.Fatal("clone changed the original choice")
	}
	g.BouncePermanentToHand(g.FindPermanent(first.ID()))
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.ChosenPlayer != a.PlayerID() {
		t.Fatal("returned object retained its old choice")
	}
}

func TestChooseOpponentAsEntersDeclaration(t *testing.T) {
	card := NewInstant("Invalid", "", WithAbility(ChooseOpponentAsEnters()))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("entry choice on a nonpermanent was accepted")
	}
}
