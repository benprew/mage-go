package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestControllerLifeAtLeastStartingLifeStateAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	a.SetStartingLife(30)
	b.SetStartingLife(15)
	a.SetLife(30)
	b.SetLife(15)
	g := NewGame(a, b)
	condition := ControllerLifeAtLeastStartingLife()
	ctxA := StateContext{ControllerID: a.PlayerID()}
	ctxB := StateContext{ControllerID: b.PlayerID()}
	if !condition.checkState(g, ctxA) || !condition.checkState(g, ctxB) {
		t.Fatal("current life at the starting total did not match")
	}
	a.SetLife(29)
	b.SetLife(16)
	if condition.checkState(g, ctxA) || !condition.checkState(g, ctxB) {
		t.Fatal("condition did not compare each controller with its own starting total")
	}
	if condition.checkState(g, StateContext{ControllerID: uuid.Nil}) || condition.checkState(g, StateContext{ControllerID: uuid.New()}) {
		t.Fatal("missing controller matched")
	}
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).SetLife(31)
	if !condition.checkState(clone, ctxA) || condition.checkState(g, ctxA) {
		t.Fatal("clone did not keep independent current life")
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" {
		t.Fatalf("inspection = %+v", got)
	}
}

func TestControllerLifeAtLeastStartingLifeRegistration(t *testing.T) {
	condition := ControllerLifeAtLeastStartingLife()
	card := NewEnchantment("Life watcher", "", WithAbility(Static(While(condition,
		ModifyEachPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Subject", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	checkPower := func(want int) {
		t.Helper()
		g.ApplyContinuousEffects()
		if got := g.FindPermanent(creature.ID()).CurrentPower(g); got != want {
			t.Fatalf("current power = %d, want %d", got, want)
		}
	}
	checkPower(3)
	a.SetLife(19)
	checkPower(2)
	a.SetLife(20)
	checkPower(3)
}
