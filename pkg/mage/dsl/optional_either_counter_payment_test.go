package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOptionalEitherFixedCounterEngineDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Synthetic optional alternative", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(mage.OptionalCost(dsl.EitherCost(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 2))), "Pay counters?")),
		dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a := &optionalXAcceptPlayer{mage.NewBasePlayer("A")}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(card.ID()).AddCounter(core.Charge, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()).Counters[core.Charge] != 1 {
		t.Fatal("activation did not pay selected counters")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	if g.FindPermanent(card.ID()).Counters[core.Charge] != 1 {
		t.Fatal("copy paid the cost again")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("resolution crossed clone boundary")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("activation and copy did not resolve")
	}
}

func TestOptionalEitherFixedCounterDeclarationValidation(t *testing.T) {
	for _, cost := range []dsl.Cost{
		mage.OptionalCost(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, -1)), "Pay?"),
		mage.OptionalCost(dsl.EitherCost(dsl.EitherCost(dsl.RemoveCountersCost(core.NumCounters, 1))), "Pay?"),
		mage.OptionalCost(dsl.EitherCost(), "Pay?"),
		dsl.OptionalCostInto(dsl.EitherCost(dsl.RemoveCountersCostInto(core.Charge, 1, dsl.BindNumber("inner"))), "Pay?", dsl.BindNumber("paid")),
	} {
		card := dsl.NewArtifact("Invalid optional alternative", "", dsl.WithAbility(dsl.Activate(dsl.Costs(cost), dsl.GainLife(dsl.You, dsl.Fixed(1)))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestOptionalEitherXCounterEngineDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Synthetic optional X alternative", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.Mana("{X}"), mage.OptionalCost(dsl.EitherCost(dsl.EitherCost(dsl.RemoveXCountersFromSourceCost(core.Charge))), "Pay counters?")),
		dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a := &optionalXAcceptPlayer{mage.NewBasePlayer("A")}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(card.ID()).AddCounter(core.Charge, 3)
	a.ManaPool().Add(core.Red, 2)
	g.SetXValue(2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()).Counters[core.Charge] != 1 {
		t.Fatal("activation did not pay selected counters")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	if g.FindPermanent(card.ID()).Counters[core.Charge] != 1 {
		t.Fatal("copy paid the cost again")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("resolution crossed clone boundary")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("activation and copy did not resolve")
	}
}

func TestOptionalEitherXCounterDeclarationValidation(t *testing.T) {
	for _, cost := range []dsl.Cost{
		mage.OptionalCost(dsl.EitherCost(dsl.EitherCost(dsl.RemoveXCountersFromSourceCost(core.NumCounters))), "Pay?"),
		mage.OptionalCost(dsl.EitherCost(), "Pay?"),
		dsl.OptionalCostInto(dsl.EitherCost(dsl.RemoveXCountersFromSourceCost(core.Charge)), "Pay?", dsl.BindNumber("paid")),
	} {
		card := dsl.NewArtifact("Invalid optional X alternative", "", dsl.WithAbility(dsl.Activate(dsl.Costs(cost), dsl.GainLife(dsl.You, dsl.Fixed(1)))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}
