package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestEitherOptionalFixedCounterPublicDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Synthetic alternative optional fixed", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.EitherCost(dsl.OptionalCostInto(dsl.RemoveCountersCost(core.Charge, 2), "Pay counters?", dsl.BindNumber("paid")))),
		dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a := &optionalXAcceptPlayer{mage.NewBasePlayer("A")}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(card.ID()).AddCounter(core.Charge, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()).Counters[core.Charge] != 0 {
		t.Fatal("optional counters were not paid at activation")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("resolution crossed clone boundary")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("copied activation did not resolve")
	}
}

func TestEitherOptionalFixedCounterResultDoesNotDominate(t *testing.T) {
	paid := dsl.BindNumber("paid")
	card := dsl.NewArtifact("Invalid optional result", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.EitherCost(dsl.OptionalCostInto(dsl.RemoveCountersCost(core.Charge, 2), "Pay counters?", paid))),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("alternative result escaped its branch")
	}
}
