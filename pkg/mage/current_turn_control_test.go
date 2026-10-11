package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCurrentTurnControlHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 2, 3)
	p := g.PutOnBattlefield(card, a.PlayerID())
	if ControlledSinceCurrentTurnStart.Match(p, g) {
		t.Fatal("new entry qualified")
	}
	g.SetActivePlayerIndex(1)
	g.SetTurn(2)
	if !ControlledSinceCurrentTurnStart.Match(p, g) || p.ControlledSinceTurnStart(g) {
		t.Fatal("current turn confused with controller's most recent turn")
	}
	clone := g.Clone()
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if ControlledSinceCurrentTurnStart.Match(g.FindPermanent(p.ID()), g) {
		t.Fatal("control loss and regain erased interruption")
	}
	if !ControlledSinceCurrentTurnStart.Match(clone.FindPermanent(p.ID()), clone) {
		t.Fatal("control change changed clone history")
	}
	g.SetTurn(3)
	p = g.FindPermanent(p.ID())
	if !ControlledSinceCurrentTurnStart.Match(p, g) {
		t.Fatal("extra turn did not establish current-turn control")
	}
	g.PhaseOut(p.ID())
	if ControlledSinceCurrentTurnStart.Match(g.FindPermanentIncludingPhased(p.ID()), g) {
		t.Fatal("phased subject qualified")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	p = g.FindPermanent(p.ID())
	if !ControlledSinceCurrentTurnStart.Match(p, g) {
		t.Fatal("phasing changed control history")
	}
	g.RemoveFromBattlefield(p)
	p = g.PutOnBattlefield(card, a.PlayerID())
	if ControlledSinceCurrentTurnStart.Match(p, g) {
		t.Fatal("returned object inherited history")
	}
	g.SetRandomResults([]int{7})
	for range 3 {
		ControlledSinceCurrentTurnStart.Match(p, g)
	}
	if g.RandIntn(10) != 7 || p.Tapped {
		t.Fatal("query changed state")
	}
}
