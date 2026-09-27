package mage

import "testing"

func TestUniqueHighestLifeStateAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := HasUniqueHighestLife()
	ctx := StateContext{ControllerID: a.PlayerID()}
	if condition.checkState(g, ctx) {
		t.Fatal("tied life totals have no unique leader")
	}
	g.PlayerGainLife(a, 1)
	if !condition.checkState(g, ctx) || !condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) {
		t.Fatal("unique leader depended on the ability controller")
	}
	clone := g.Clone()
	clone.PlayerGainLife(clone.GetPlayer(b.PlayerID()), 2)
	if !condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone did not read independent life totals")
	}
	clone.PlayerLoseLife(clone.GetPlayer(b.PlayerID()), 1)
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("tie or clone changed original state")
	}
}

func TestUniqueHighestLifeInterveningIfAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := HasUniqueHighestLife()
	source := NewCreature("Life watcher", "", 2, 2, WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "one player has the highest life total" {
		t.Fatalf("inspection = %+v", got)
	}
}
