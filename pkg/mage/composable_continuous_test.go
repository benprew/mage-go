package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableContinuousCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2)
	first.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	effect := Until(EndOfTurn, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(2), Fixed(1)))
	if err := ApplyEffect(g, effect, first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	later := NewCreature("Later", "", 2, 2)
	later.SetOwner(a.PlayerID())
	g.PutOnBattlefield(later, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).CurrentPower(g) != 4 || g.FindPermanent(later.ID()).CurrentPower(g) != 2 {
		t.Fatal("temporary membership was not captured")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).CurrentPower(g) != 2 {
		t.Fatal("temporary effect followed a returned card")
	}
}

func TestComposableStaticMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lord := NewCreature("Lord", "", 2, 2, WithAbility(Static(While(Controls(IsCreature), ModifyEachPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(1))))))
	lord.SetOwner(a.PlayerID())
	g.PutOnBattlefield(lord, a.PlayerID())
	later := NewCreature("Later", "", 2, 2)
	later.SetOwner(a.PlayerID())
	g.PutOnBattlefield(later, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(later.ID()).CurrentPower(g) != 3 {
		t.Fatal("static membership did not include later entry")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(lord.ID()))
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(later.ID()).CurrentPower(clone) != 2 || g.FindPermanent(later.ID()).CurrentPower(g) != 3 {
		t.Fatal("static clone isolation failed")
	}
}
