package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestXActivationPaymentReduction(t *testing.T) {
	for _, tc := range []struct {
		cost               string
		x, reduction, want int
	}{
		{"{X}", 3, 2, 1}, {"{X}", 0, 2, 0}, {"{X}", 1, 2, 1}, {"{X}{X}", 3, 2, 4}, {"{X}{U}", 3, 2, 2}, {"{2}{X}", 3, 2, 3}, {"{X}{U}", 0, 2, 1},
	} {
		t.Run(fmt.Sprintf("%s/%d", tc.cost, tc.x), func(t *testing.T) {
			g, a, _ := newActivationDivisionGame()
			source := g.PutOnBattlefield(NewArtifact("Variable", "", WithAbility(Activate(Costs(ManaCostOf(tc.cost), Tap()), GainLifeFor(You, ValueAtResolution(XValue()))))), a.PlayerID())
			g.PutOnBattlefield(NewEnchantment("Reducer", "", WithStaticAbility(FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(game *Game, _ uuid.UUID) error {
				game.AddActivationCostReduction(source.ID(), tc.reduction)
				return nil
			}))), a.PlayerID())
			mana := core.ParseManaCost(tc.cost)
			a.ManaPool().Add(core.Blue, mana.Blue)
			a.ManaPool().Add(core.Colorless, tc.want-mana.Blue)
			g.SetXValue(tc.x)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped || g.StackPeek().XValue != tc.x {
				t.Fatal("wrong reduced payment or X")
			}
			g.ResolveStack()
			if a.Life() != 20+tc.x {
				t.Fatal("reduction changed announced X")
			}
		})
	}
}

func TestXActivationPaymentAggregateReduction(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint(legacy), func(t *testing.T) {
			g, a, _ := newActivationDivisionGame()
			variable := ManaCostOf("{X}")
			if legacy {
				variable = XManaCost()
			}
			source := g.PutOnBattlefield(NewArtifact("Variable", "", WithAbility(Activate(Costs(variable, ManaCostOf("{X}"), Tap()), GainLifeFor(You, Fixed(1))))), a.PlayerID())
			g.PutOnBattlefield(NewEnchantment("Reducer", "", WithStaticAbility(FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(game *Game, _ uuid.UUID) error { game.AddActivationCostReduction(source.ID(), 2); return nil }))), a.PlayerID())
			g.SetXValue(3)
			a.ManaPool().Add(core.Colorless, 4)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("reduction applied more than once to total cost")
			}
		})
	}
}

func TestXActivationPaymentNegativeRollback(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	source := g.PutOnBattlefield(NewArtifact("Variable", "", WithAbility(Activate(Costs(ManaCostOf("{X}"), Tap()), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	g.SetXValue(-1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("negative announced X accepted")
	}
	if g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
		t.Fatal("negative X changed live state")
	}
}
