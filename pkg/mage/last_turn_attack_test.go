package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLastTurnAttackIdentity(t *testing.T) {
	for _, change := range []string{"none", "current-turn", "empty-turn", "other-turn", "control", "phase", "phase-cycle", "return", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			condition := PermanentAttackedDuringLastTurn(This, You)
			ctx := StateContext{SourceID: p.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
			if condition.checkState(g, ctx) {
				t.Fatal("history exists before a turn completes")
			}
			g.trackers.Turn.RecordAttacked(p.ID())
			g.trackers.Turn.RecordAttackedIncarnation(p.ID(), p.incarnationID)
			if condition.checkState(g, ctx) {
				t.Fatal("current turn replaced last turn")
			}
			g.trackers.Turn.ResetForNewTurn(a.PlayerID())
			clone := g.Clone()
			want := true
			switch change {
			case "current-turn":
				g.trackers.Turn.RecordAttacked(p.ID())
				g.trackers.Turn.RecordAttackedIncarnation(p.ID(), p.incarnationID)
			case "empty-turn":
				g.trackers.Turn.ResetForNewTurn(a.PlayerID())
				want = false
			case "other-turn":
				g.trackers.Turn.ResetForNewTurn(b.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				ctx.ControllerID = b.PlayerID()
				want = false
			case "phase":
				g.PhaseOut(p.ID())
				want = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
				ctx.SourceIncarnation = p.incarnationID
				want = false
			case "clone":
				clone.trackers.Turn.ResetForNewTurn(a.PlayerID())
			}
			if condition.checkState(g, ctx) != want {
				t.Fatal("wrong live history after", change)
			}
			originalCtx := ctx
			originalCtx.ControllerID = a.PlayerID()
			originalCtx.SourceIncarnation = clone.FindPermanent(card.ID()).incarnationID
			if condition.checkState(clone, originalCtx) != (change != "clone") {
				t.Fatal("clone archive is not independent")
			}
			if g.AttackedDuringLastTurnIncarnation(a.PlayerID(), card.ID(), uuid.New()) {
				t.Fatal("wrong incarnation matches")
			}
		})
	}
}

func TestLastTurnAttackConditionalRule(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 2, 4, WithAbility(Static(CantAttackIf(This, PermanentAttackedDuringLastTurn(This, You)))))
	p := g.PutOnBattlefield(c, a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	if !p.CanDeclareAsAttacker(g) {
		t.Fatal("false condition prevented attack")
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("legal attack failed")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.UntapPermanent(g.FindPermanent(p.ID()))
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("last-turn attack was permitted")
	}
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("restriction removed an existing attacker")
	}
	clone := g.Clone()
	clone.trackers.Turn.ResetForNewTurn(a.PlayerID())
	if !clone.FindPermanent(p.ID()).CanDeclareAsAttacker(clone) || g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("rule clone state")
	}
	g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
	if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("removed source ability still restricts")
	}
}

func TestLastTurnAttackDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewCreature("Good", "", 2, 4, WithAbility(Static(CantAttackIf(This, PermanentAttackedDuringLastTurn(This, You)))))},
		{Name: "nil subject", Card: NewCreature("Bad", "", 2, 4, WithAbility(Static(CantAttackIf(nil, PermanentAttackedDuringLastTurn(This, You))))), WantErr: true},
		{Name: "nil condition", Card: NewCreature("Bad", "", 2, 4, WithAbility(Static(CantAttackIf(This, nil)))), WantErr: true},
		{Name: "nil history subject", Card: NewCreature("Bad", "", 2, 4, WithAbility(Static(CantAttackIf(This, PermanentAttackedDuringLastTurn(nil, You))))), WantErr: true},
		{Name: "nil player", Card: NewCreature("Bad", "", 2, 4, WithAbility(Static(CantAttackIf(This, PermanentAttackedDuringLastTurn(This, nil))))), WantErr: true},
		{Name: "unbound", Card: NewCreature("Bad", "", 2, 4, WithAbility(Static(CantAttackIf(This, PermanentAttackedDuringLastTurn(DeclarePermanentTarget("missing", IsCreature), You))))), WantErr: true},
	})
}

func TestLastTurnAttackMultipleCombatsAndPlayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 2, 8)
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	for range 2 {
		g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
		g.UntapPermanent(g.FindPermanent(p.ID()))
		g.GetCombat().Reset()
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
		if !g.GetCombat().IsAttacking(p.ID()) {
			t.Fatal("combat did not record attack")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	if !g.AttackedDuringLastTurnIncarnation(a.PlayerID(), p.ID(), p.incarnationID) {
		t.Fatal("multiple-combat turn archive")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Cleanup)
	if !g.AttackedDuringLastTurnIncarnation(a.PlayerID(), p.ID(), p.incarnationID) || g.AttackedDuringLastTurnIncarnation(b.PlayerID(), p.ID(), p.incarnationID) {
		t.Fatal("other player's empty turn changed archive")
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Cleanup)
	if g.AttackedDuringLastTurnIncarnation(a.PlayerID(), p.ID(), p.incarnationID) {
		t.Fatal("own empty turn did not clear history")
	}
}
