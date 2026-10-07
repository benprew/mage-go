package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestContinuousControlUsesControllersTurn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewArtifact("Subject", "")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	p.GrantBaseAttr(core.Haste)
	if p.ControlledSinceTurnStart(g) {
		t.Fatal("entry after turn start qualified")
	}
	g.SetActivePlayerIndex(1)
	g.SetTurn(2)
	if p.ControlledSinceTurnStart(g) {
		t.Fatal("opponent turn restored continuous control")
	}
	clone := g.Clone()
	clone.SetTurn(3)
	if clone.FindPermanent(p.ID()).ControlledSinceTurnStart(clone) {
		t.Fatal("opponent extra turn restored continuous control")
	}
	clone.SetActivePlayerIndex(0)
	clone.SetTurn(4)
	if !clone.FindPermanent(p.ID()).ControlledSinceTurnStart(clone) || p.ControlledSinceTurnStart(g) {
		t.Fatal("own turn or clone history incorrect")
	}
	g.SetActivePlayerIndex(0)
	g.SetTurn(3)
	if !p.ControlledSinceTurnStart(g) {
		t.Fatal("own turn did not restore continuous control")
	}
	g.PhaseOut(p.ID())
	g.SetActivePlayerIndex(1)
	g.SetTurn(4)
	g.PhaseIn([]uuid.UUID{p.ID()})
	if !g.FindPermanent(p.ID()).ControlledSinceTurnStart(g) {
		t.Fatal("phasing erased control history")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.SetActivePlayerIndex(0)
	g.SetTurn(5)
	if g.FindPermanent(p.ID()).ControlledSinceTurnStart(g) {
		t.Fatal("new controller used another player's turn")
	}
	g.SetActivePlayerIndex(1)
	g.SetTurn(6)
	if !g.FindPermanent(p.ID()).ControlledSinceTurnStart(g) {
		t.Fatal("new controller's own turn did not qualify")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(c.ID())
	returned := g.PutOnBattlefield(c, b.PlayerID())
	if returned.ControlledSinceTurnStart(g) {
		t.Fatal("returned incarnation inherited control history")
	}
}

func TestContinuousControlSkippedTurn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	b.AddToLibrary(NewArtifact("Draw", ""))
	g.SkipNextTurnFor(a.PlayerID())
	g.Run(0, core.Untap, 2)
	if g.turns.turnStarts[0] != 0 || g.turns.turnStarts[1] != 2 {
		t.Fatal("native skipped turn changed observed history")
	}
}

func TestContinuousControlNativeTurnHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.RunStepWithPriority(core.Untap)
	p := g.PutOnBattlefield(NewArtifact("Subject", ""), a.PlayerID())
	g.turns.SetActivePlayerIndex(1)
	g.turns.IncrementTurn()
	g.RunStepWithPriority(core.Upkeep)
	if p.ControlledSinceTurnStart(g) || g.MostRecentTurnStart(a.PlayerID()) != 1 {
		t.Fatal("native opponent turn lost history")
	}
	g.turns.IncrementTurn()
	g.RunStepWithPriority(core.Upkeep)
	if p.ControlledSinceTurnStart(g) {
		t.Fatal("native opponent extra turn granted control permission")
	}
	g.turns.SetActivePlayerIndex(0)
	g.turns.IncrementTurn()
	g.RunStepWithPriority(core.Upkeep)
	if !p.ControlledSinceTurnStart(g) || g.MostRecentTurnStart(b.PlayerID()) != 3 {
		t.Fatal("skipped untap did not record the turn start")
	}
	clone := g.Clone()
	clone.turns.IncrementTurn()
	clone.RunStepWithPriority(core.Upkeep)
	clone.turns.SetActivePlayerIndex(1)
	clone.turns.IncrementTurn()
	if clone.MostRecentTurnStart(a.PlayerID()) != 5 || g.MostRecentTurnStart(a.PlayerID()) != 4 {
		t.Fatal("native history was shared by clones")
	}
}
