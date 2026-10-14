package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPTSwitchDefinition(t *testing.T) {
	card := NewCreature("Switch", "", 2, 5, WithAbility(Activate(Costs(), Until(core.EndOfTurn, SwitchPowerToughness(This)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	bad := NewCreature("Bad", "", 2, 5, WithAbility(Activate(Costs(), Until(core.EndOfTurn, SwitchPowerToughness(nil)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("accepted missing switch subject")
	}
	for _, mod := range []ContinuousModification{ModifyPT(This, nil, Fixed(1)), SetBasePowerToughness(This, Fixed(1), nil)} {
		bad := NewCreature("Bad", "", 2, 5, WithAbility(Activate(Costs(), Until(core.EndOfTurn, mod))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted missing numeric operand")
		}
	}
}

func TestPTSwitchLayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 3), a.PlayerID())
	apply := func(mods ...ContinuousModification) {
		t.Helper()
		if err := ApplyEffect(g, Until(core.EndOfTurn, mods...), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	check := func(power, toughness int) {
		t.Helper()
		for range 3 {
			g.ApplyContinuousEffects()
			q := g.FindPermanent(p.ID())
			if q.CurrentPower(g) != power || q.CurrentToughness(g) != toughness {
				t.Fatalf("got %d/%d, want %d/%d", q.CurrentPower(g), q.CurrentToughness(g), power, toughness)
			}
		}
	}
	apply(SwitchPowerToughness(This), ModifyPT(This, Fixed(0), Fixed(1)))
	check(4, 1)
	apply(ModifyPT(This, Fixed(5), Fixed(0)))
	check(4, 6)
	apply(SetBasePowerToughness(This, Fixed(2), Fixed(7)))
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
	check(9, 8)
	apply(SwitchPowerToughness(This))
	check(8, 9)
	g.doCleanupActions()
	check(2, 4)
}
