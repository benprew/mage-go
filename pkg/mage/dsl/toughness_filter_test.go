package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestHasToughnessGTE(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	c := dsl.NewCreature("Recipient", "", 2, 3)
	g.PutOnBattlefield(c, b.PlayerID())
	p := g.FindPermanent(c.ID())
	filter := dsl.And(dsl.IsCreature, dsl.HasToughnessGTE(4))
	if filter.Match(p, g) {
		t.Fatal("toughness 3 matched threshold 4")
	}
	for _, delta := range []int{1, 1, -2} {
		body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.This, dsl.Fixed(0), dsl.Fixed(delta)))
		if err := mage.ApplyEffect(g, body, c.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := p.CurrentToughness(g) >= 4
		if filter.Match(p, g) != want {
			t.Fatalf("filter did not use current toughness %d", p.CurrentToughness(g))
		}
	}
	if !dsl.HasToughnessGTE(-1).Match(p, g) {
		t.Fatal("negative threshold rejected")
	}
	target := dsl.TargetPermanent("tough creature", filter)
	spell := dsl.NewInstant("Filtered mode", "", dsl.WithAbility(dsl.SpellModes(
		dsl.ModeOf("Destroy", dsl.Destroy(target), dsl.WithTargets(target)),
		dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1))),
	)))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}
