package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceAttackedOrBlockedThisTurnIdentity(t *testing.T) {
	for _, eventType := range []EventType{EvtDeclaredAttacker, EvtDeclaredBlocker} {
		t.Run(eventType.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("History", "", 2, 2)
			g.PutOnBattlefield(card, a.PlayerID())
			p := g.FindPermanent(card.ID())
			ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
			condition := SourceAttackedOrBlockedThisTurnCondition()
			if condition.checkState(g, ctx) {
				t.Fatal("history exists before declaration")
			}
			g.FireEvent(GameEvent{Type: eventType, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
			if !condition.checkState(g, ctx) {
				t.Fatal("declaration was not recorded")
			}
			clone := g.Clone()
			if !condition.checkState(clone, ctx) {
				t.Fatal("clone lost history")
			}
			g.FindPermanent(card.ID()).PhasedOut = true
			if condition.checkState(g, ctx) {
				t.Fatal("phased source matched")
			}
			g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
			g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			g.PutOnBattlefield(card, a.PlayerID())
			if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID(), SourceIncarnation: g.FindPermanent(card.ID()).incarnationID}) {
				t.Fatal("new incarnation inherited history")
			}
			if !condition.checkState(clone, ctx) {
				t.Fatal("clone changed with original")
			}
			clone.trackers.Turn.ResetForNewTurn(a.PlayerID())
			if condition.checkState(clone, ctx) {
				t.Fatal("history survived turn reset")
			}
		})
	}
}

func TestSourceAttackedOrBlockedThisTurnDeclaration(t *testing.T) {
	condition := SourceAttackedOrBlockedThisTurnCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndStep, IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "source attacked or blocked this turn" {
		t.Fatalf("metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceAttackedOrBlockedThisTurnTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndStep,
		IfState(SourceAttackedOrBlockedThisTurnCondition(), GainLifeFor(You, Fixed(1))))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("trigger did not use clone history")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("returned source used the old history")
	}
}
