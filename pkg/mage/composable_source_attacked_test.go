package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceAttackedThisTurnState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	p := g.FindPermanent(card.ID())
	ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
	condition := SourceAttackedThisTurnCondition()
	if condition.checkState(g, ctx) {
		t.Fatal("source matched before attacking")
	}
	g.trackers.Turn.RecordAttacked(card.ID())
	g.trackers.Turn.RecordAttackedIncarnation(card.ID(), p.incarnationID)
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("declared attack was not observed in game and clone")
	}
	g.FindPermanent(card.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased-out source matched")
	}
	g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) {
		t.Fatal("returned source inherited attack history")
	}
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	if condition.checkState(g, StateContext{SourceID: card.ID(), SourceIncarnation: g.FindPermanent(card.ID()).incarnationID}) {
		t.Fatal("attack history survived turn reset")
	}
}

func TestSourceAttackedThisTurnDeclaration(t *testing.T) {
	condition := SourceAttackedThisTurnCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Attacks(This), IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "source attacked this turn" {
		t.Fatalf("condition metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceAttackedThisTurnTriggerRecheck(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Attacks(This),
		IfState(SourceAttackedThisTurnCondition(), GainLifeFor(You, Fixed(1))))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	incarnation := g.FindPermanent(card.ID()).incarnationID
	g.trackers.Turn.RecordAttacked(card.ID())
	g.trackers.Turn.RecordAttackedIncarnation(card.ID(), incarnation)
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("clone did not keep its own attack history")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("returned source used the old attack history")
	}
}
