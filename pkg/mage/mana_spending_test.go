package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestColoredManaSpending(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Permission", "", WithStaticAbility(ControllerManaSpending(core.White, core.Red))), a.PlayerID())
	a.ManaPool().Add(core.White, 2)
	b.ManaPool().Add(core.White, 2)
	cost := core.ManaCost{Red: 1, White: 1}
	if !g.CanAfford(a.PlayerID(), cost, nil) || g.CanAfford(b.PlayerID(), cost, nil) {
		t.Fatal("controller permission")
	}
	if a.ManaPool().Count(core.White) != 2 {
		t.Fatal("query spent mana")
	}
	if err := a.ManaPool().Pay(cost, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().LastDrainedColors[core.White] != 2 || a.ManaPool().LastDrainedColors[core.Red] != 0 {
		t.Fatal("actual colors changed")
	}
	g.PutOnBattlefield(NewArtifact("Second", "", WithStaticAbility(ControllerManaSpending(core.White, core.Blue))), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Chain", "", WithStaticAbility(ControllerManaSpending(core.Blue, core.Green))), a.PlayerID())
	a.ManaPool().Add(core.White, 3)
	if !g.CanAfford(a.PlayerID(), core.ManaCost{Red: 1, Blue: 1, Green: 1}, nil) {
		t.Fatal("permissions did not combine")
	}
	g.PutOnBattlefield(NewArtifact("Cycle", "", WithStaticAbility(ControllerManaSpending(core.Blue, core.White))), a.PlayerID())
	if !g.CanAfford(a.PlayerID(), core.ManaCost{Red: 1, Blue: 1, Green: 1}, nil) || g.CanAfford(a.PlayerID(), core.ManaCost{Black: 1}, nil) {
		t.Fatal("cycle supplied unrelated mana")
	}
	clone := g.Clone()
	a.ManaPool().Clear()
	a.ManaPool().AddRestricted(core.White, 1, core.ArtifactSpellsOnly{})
	if g.CanAfford(a.PlayerID(), core.ManaCost{Red: 1}, nil) {
		t.Fatal("permission removed restriction")
	}
	artifact := core.SpellPaymentContext{IsArtifact: true}
	if !g.CanAfford(a.PlayerID(), core.ManaCost{Red: 1}, &artifact) {
		t.Fatal("eligible restricted mana rejected")
	}
	if !clone.CanAfford(a.PlayerID(), core.ManaCost{Red: 1, Blue: 1, Green: 1}, nil) {
		t.Fatal("clone changed")
	}
}

func TestColoredManaSpendingDeclarations(t *testing.T) {
	for _, colors := range [][2]core.Color{{core.Colorless, core.Red}, {core.White, core.AnyColor}, {core.Color(99), core.Red}} {
		c := NewArtifact("Invalid", "", WithStaticAbility(ControllerManaSpending(colors[0], colors[1])))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid color accepted")
		}
	}
}

func TestColoredManaSpendingActivationAndHybrid(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Permission", "", WithStaticAbility(ControllerManaSpending(core.White, core.Red))), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Converter", "", WithAbility(Activate(Costs(ManaCostOf("{R}")), AddMana(core.Blue, 1)))), a.PlayerID())
	a.ManaPool().Add(core.White, 1)
	if !g.CanAfford(a.PlayerID(), core.ManaCost{Blue: 1}, nil) || a.ManaPool().Count(core.White) != 1 {
		t.Fatal("conversion query")
	}
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{Blue: 1}); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Blue) != 1 || a.ManaPool().Count(core.White) != 0 || g.StackSize() != 0 {
		t.Fatal("activation payment")
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(core.White, 2)
	hybrid := core.ManaCost{White: 1, Hybrid: []core.HybridSymbol{{A: core.Red, B: core.Blue}}}
	if !a.ManaPool().CanPay(hybrid, nil) {
		t.Fatal("hybrid permission")
	}
	if err := a.ManaPool().Pay(hybrid, nil); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.White, 4)
	x := core.ManaCost{HasX: true, XCount: 1, Red: 1}
	if g.MaxXValue(a.PlayerID(), x, nil) != 3 {
		t.Fatal("X bound")
	}
	g.PutOnBattlefield(NewArtifact("Opponent permission", "", WithStaticAbility(ControllerManaSpending(core.White, core.Green))), b.PlayerID())
	b.ManaPool().Add(core.White, 1)
	if !g.CanAfford(b.PlayerID(), core.ManaCost{Green: 1}, nil) || g.CanAfford(a.PlayerID(), core.ManaCost{Green: 1}, nil) {
		t.Fatal("two-player independence")
	}
}
