package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestExactStatsFilters(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	c := dsl.NewCreature("Recipient", "", 1, 1)
	p := g.PutOnBattlefield(c, b.PlayerID())
	filter := dsl.And(dsl.IsCreature, dsl.HasPowerEqual(1), dsl.HasToughnessEqual(1))
	for _, step := range []struct {
		power, toughness int
		matches          bool
	}{
		{0, 0, true}, {1, 0, false}, {-1, 1, false}, {0, -1, true}, {-2, -2, false},
	} {
		body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.This, dsl.Fixed(step.power), dsl.Fixed(step.toughness)))
		if err := mage.ApplyEffect(g, body, c.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if got := filter.Match(p, g); got != step.matches {
			t.Fatalf("current %d/%d: match = %v, want %v", p.CurrentPower(g), p.CurrentToughness(g), got, step.matches)
		}
	}
	if !dsl.HasPowerEqual(-1).Match(p, g) || !dsl.HasToughnessEqual(-1).Match(p, g) {
		t.Fatal("negative current characteristics did not match")
	}
	target := dsl.TargetPermanent("creature", filter)
	card := dsl.NewLand("Booster", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.TapCost(dsl.This)),
		dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.Fixed(1), dsl.Fixed(2))),
		dsl.WithTargets(target),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
