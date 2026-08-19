package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestEventSourceWasColor_LivePermanent(t *testing.T) {
	g, a, _ := randomTestGame()
	card := NewCreature("Color Test Bear", "{1}{G}", 2, 2)
	card.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(card, a.PlayerID())
	evt := &GameEvent{SourceID: perm.ID()}

	if !((EventSourceWasColor{Color: Green}).CheckTriggerCond(evt, g, perm.ID(), a.PlayerID())) {
		t.Error("green live permanent did not match green")
	}
	if (EventSourceWasNotColor{Color: Green}).CheckTriggerCond(evt, g, perm.ID(), a.PlayerID()) {
		t.Error("green live permanent matched not green")
	}
	if !((EventSourceWasNotColor{Color: Black}).CheckTriggerCond(evt, g, perm.ID(), a.PlayerID())) {
		t.Error("green live permanent did not match not black")
	}
}

func TestEventSourceWasColor_UsesRecoloredPermanentLKI(t *testing.T) {
	g, a, _ := randomTestGame()
	card := NewCreature("Recolored Test Bear", "{1}{G}", 2, 2)
	card.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(card, a.PlayerID())
	black := []Color{Black}
	perm.ColorOverride = &black
	evt := &GameEvent{SourceID: perm.ID()}

	g.RemoveFromBattlefield(perm)

	if !((EventSourceWasColor{Color: Black}).CheckTriggerCond(evt, g, perm.ID(), a.PlayerID())) {
		t.Error("departed permanent did not retain its black color override in LKI")
	}
	if (EventSourceWasColor{Color: Green}).CheckTriggerCond(evt, g, perm.ID(), a.PlayerID()) {
		t.Error("departed recolored permanent matched its printed green color")
	}
	if (EventSourceWasNotColor{Color: Black}).CheckTriggerCond(evt, g, perm.ID(), a.PlayerID()) {
		t.Error("departed black permanent matched not black")
	}
}

func TestEventSourceWasColor_MissingObjectDoesNotMatch(t *testing.T) {
	g, a, _ := randomTestGame()
	evt := &GameEvent{}

	if (EventSourceWasColor{Color: Black}).CheckTriggerCond(evt, g, evt.SourceID, a.PlayerID()) {
		t.Error("missing object matched a color")
	}
	if (EventSourceWasNotColor{Color: Black}).CheckTriggerCond(evt, g, evt.SourceID, a.PlayerID()) {
		t.Error("missing object matched not a color")
	}
}
