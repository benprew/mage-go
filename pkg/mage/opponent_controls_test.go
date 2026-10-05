package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestOpponentControlsState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := OpponentControls(HasSubType("Island"))
	ctx := StateContext{ControllerID: a.PlayerID()}
	island := NewLand("Island", WithSubTypes("Island"))
	island.SetOwner(a.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("empty battlefield matched")
	}
	p := g.PutOnBattlefield(island, b.PlayerID())
	for _, active := range []int{0, 1} {
		g.SetActivePlayerIndex(active)
		if !condition.checkState(g, ctx) || condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) {
			t.Fatal("condition used turn or ownership instead of control")
		}
	}
	for _, id := range []uuid.UUID{uuid.Nil, uuid.New()} {
		if condition.checkState(g, StateContext{ControllerID: id}) {
			t.Fatal("unknown controller matched")
		}
	}
	phased := g.PhaseOut(p.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("phased Island matched")
	}
	g.PhaseIn(phased)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone changed original membership")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("departure used last known control")
	}
	g.PutOnBattlefield(island, a.PlayerID())
	if condition.checkState(g, ctx) || !condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) {
		t.Fatal("return retained old control")
	}
}

func TestOpponentControlsDefinition(t *testing.T) {
	condition := OpponentControls(IsLand)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "static", Card: NewCreature("Static", "", 1, 1, WithAbility(Static(While(condition, ModifyPT(This, Fixed(1), Fixed(1)))))),
			Inspect: func(t *testing.T) {
				if info := InspectDefinition(condition); info.Role != "state-condition" || info.Text == "" {
					t.Fatalf("inspection = %+v", info)
				}
			}},
		{Name: "resolution", Card: NewInstant("Spell", "", WithAbility(Spell(IfState(condition, GainLifeFor(You, Fixed(1))))))},
	})
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := IfState(condition, GainLifeFor(You, Fixed(1)))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("resolution did not read current membership")
	}
}
