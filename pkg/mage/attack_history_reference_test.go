package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackHistoryReferenceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Attacker", "", 2, 4)
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	if !AttackedThisTurn.Match(p, g) || DidntAttackThisTurn.Match(p, g) {
		t.Fatal("declared attack was not recorded")
	}
	inc := p.incarnationID
	binding := BindPermanent("original")
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
	if err := Snapshot(This, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if AttackedThisTurn.Match(returned, g) || !DidntAttackThisTurn.Match(returned, g) {
		t.Fatal("returned object inherited attack history")
	}
	returned.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{returned.ID()})
	if !g.HasAttackedThisTurnIncarnation(card.ID(), inc) || !AttackedThisTurn.Match(returned, g) {
		t.Fatal("later attack erased the old incarnation")
	}
	if !AttackedThisTurn.Match(g.zones.LKIByIncarnation(inc).Snapshot, g) {
		t.Fatal("retained information lost attack history")
	}
	if err := IfPermanentMatches(From(binding), AttackedThisTurn, GainLifeFor(You, Fixed(2))).Apply(ctx); err != nil || a.Life() != 22 {
		t.Fatal("saved reference did not use the old incarnation")
	}
	if AttackedThisTurn.Match(nil, g) || DidntAttackThisTurn.Match(nil, g) {
		t.Fatal("absent object satisfied a history filter")
	}
	clone := g.Clone()
	clone.trackers.Turn.ResetForNewTurn(a.PlayerID())
	if AttackedThisTurn.Match(clone.FindPermanent(returned.ID()), clone) || !AttackedThisTurn.Match(returned, g) {
		t.Fatal("history reset changed the original game")
	}
}

func TestAttackHistoryReferencePhasingAndCombat(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 4), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.GetCombat().AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	if AttackedThisTurn.Match(p, g) || g.PlayerAttackedThisTurn(a.PlayerID()) {
		t.Fatal("entering combat without declaration counted as an attack")
	}
	g.RemoveFromCombat(p.ID())
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	g.RemoveFromCombat(p.ID())
	g.PhaseIn(g.PhaseOut(p.ID()))
	if !AttackedThisTurn.Match(g.FindPermanent(p.ID()), g) {
		t.Fatal("combat exit or phasing erased history")
	}
	g.RunStepWithPriority(core.Cleanup)
	if AttackedThisTurn.Match(g.FindPermanent(p.ID()), g) || g.PlayerAttackedThisTurn(a.PlayerID()) {
		t.Fatal("native cleanup retained history")
	}
}

func TestAttackHistoryPlayerTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, options := range [][]PlayerTargetOption{
		{TargetPlayerAttackedThisTurn()},
		{TargetPlayerAttackedThisTurn(), TargetOpponentOnly()},
		{TargetOpponentOnly(), TargetPlayerAttackedThisTurn()},
	} {
		target := DeclarePlayerTarget("attacking player", options...)
		if len(target.Possible(a.PlayerID(), nil, g)) != 0 {
			t.Fatal("player qualified without an attack")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, PlayerID: b.PlayerID(), SourceID: uuid.New()})
		if !slices.Equal(target.Possible(a.PlayerID(), nil, g), []uuid.UUID{b.PlayerID()}) {
			t.Fatal("attack history restriction lost")
		}
		g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	}
	card := NewInstant("Invalid", "", Spell(GainLifeFor(DeclarePlayerTarget("missing", TargetPlayerAttackedThisTurn()), Fixed(1))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("undeclared target accepted")
	}
	target := DeclarePlayerTarget("history", TargetPlayerAttackedThisTurn())
	spell := NewInstant("Valid", "", Spell(GainLifeFor(target, Fixed(2)), WithTargets(target)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, PlayerID: a.PlayerID(), SourceID: uuid.New()})
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("resolution did not recheck player history")
	}
}

func TestAttackHistoryBothDeclarationPaths(t *testing.T) {
	for _, native := range []bool{false, true} {
		a := &attackPaymentIdentityPlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetTurn(1)
		p := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 4), a.PlayerID())
		a.attacker = p.ID()
		if native {
			g.RunStepWithPriority(core.DeclareAttackers)
		} else {
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
		}
		if AttackedThisTurn.Match(p, g) || g.PlayerAttackedThisTurn(a.PlayerID()) {
			t.Fatal("illegal attack changed history")
		}
		g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
		if native {
			g.RunStepWithPriority(core.DeclareAttackers)
		} else {
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
		}
		if !AttackedThisTurn.Match(g.FindPermanent(p.ID()), g) || !g.PlayerAttackedThisTurn(a.PlayerID()) {
			t.Fatal("legal attack did not change both histories")
		}
	}
}
