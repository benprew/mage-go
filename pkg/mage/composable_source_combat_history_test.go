package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceAttackedOrBlockedThisCombatHistory(t *testing.T) {
	for _, eventType := range []EventType{EvtDeclaredAttacker, EvtDeclaredBlocker} {
		t.Run(eventType.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("History", "", 2, 2)
			g.PutOnBattlefield(card, a.PlayerID())
			p := g.FindPermanent(card.ID())
			ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
			condition := SourceAttackedOrBlockedThisCombatCondition()
			g.FireEvent(GameEvent{Type: EvtBeginCombat, PlayerID: a.PlayerID()})
			if condition.checkState(g, ctx) {
				t.Fatal("history exists before declaration")
			}
			g.FireEvent(GameEvent{Type: eventType, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
			if !condition.checkState(g, ctx) {
				t.Fatal("declaration was not recorded")
			}
			clone := g.Clone()
			g.FindPermanent(card.ID()).PhasedOut = true
			if condition.checkState(g, ctx) {
				t.Fatal("phased source matched")
			}
			g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
			g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			g.PutOnBattlefield(card, a.PlayerID())
			if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) {
				t.Fatal("returned source inherited combat history")
			}
			if !condition.checkState(clone, ctx) {
				t.Fatal("clone lost independent history")
			}
			clone.RunStepWithPriority(EndCombat)
			if condition.checkState(clone, ctx) {
				t.Fatal("history survived end of combat")
			}
			clone.FireEvent(GameEvent{Type: eventType, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
			clone.FireEvent(GameEvent{Type: EvtBeginCombat, PlayerID: a.PlayerID()})
			if condition.checkState(clone, ctx) {
				t.Fatal("history survived next combat")
			}
			if !SourceAttackedOrBlockedThisTurnCondition().checkState(clone, ctx) {
				t.Fatal("combat reset cleared turn history")
			}
		})
	}
}

func TestSourceAttackedOrBlockedThisCombatDeclaration(t *testing.T) {
	condition := SourceAttackedOrBlockedThisCombatCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndOfCombat, IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(condition); meta.Role != "state-condition" || meta.Text != "source attacked or blocked this combat" {
		t.Fatalf("metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceAttackedOrBlockedThisCombatEndTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndOfCombat,
		IfState(SourceAttackedOrBlockedThisCombatCondition(), GainLifeFor(You, Fixed(1))))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtBeginCombat, PlayerID: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID()})
	g.RunStepWithPriority(EndCombat)
	if a.Life() != 21 {
		t.Fatalf("end of combat life = %d", a.Life())
	}
	ctx := StateContext{SourceID: card.ID(), SourceIncarnation: g.FindPermanent(card.ID()).incarnationID}
	if SourceAttackedOrBlockedThisCombatCondition().checkState(g, ctx) {
		t.Fatal("combat history remained after end of combat")
	}
	if !SourceAttackedOrBlockedThisTurnCondition().checkState(g, ctx) {
		t.Fatal("turn history was cleared at end of combat")
	}
}
