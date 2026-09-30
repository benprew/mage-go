package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOptionalCounterAlternativeResultPublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid")
	cost := dsl.OptionalCostInto(dsl.EitherCost(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 2))), "Pay counters?", paid)
	card := dsl.NewArtifact("Synthetic optional counter result", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(cost), dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a := &optionalXAcceptPlayer{mage.NewBasePlayer("A")}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(card.ID()).AddCounter(core.Charge, 2)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.FindPermanent(card.ID()).Counters[core.Charge] != 0 {
		t.Fatal("payment did not occur at activation")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("result crossed clone boundary")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("acceptance was overwritten by a later unaffordable activation or copy")
	}
}

func TestOptionalCounterAlternativeResultValidation(t *testing.T) {
	paid := dsl.BindNumber("paid")
	wrap := func(inner dsl.Cost) dsl.Cost { return dsl.OptionalCostInto(inner, "Pay?", paid) }
	for _, cost := range []dsl.Cost{
		wrap(dsl.EitherCost()),
		wrap(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, -1))),
		wrap(dsl.EitherCost(dsl.RemoveCountersCost(core.NumCounters, 1))),
		wrap(dsl.EitherCost(dsl.RemoveXCountersFromSourceCost(core.Charge), dsl.RemoveCountersCost(core.Charge, 1))),
		wrap(dsl.EitherCost(dsl.RemoveCountersCostInto(core.Charge, 1, dsl.BindNumber("inner")))),
		wrap(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 1), dsl.LifePayCost(1))),
		dsl.OptionalCostInto(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 1)), "Pay?", nil),
		dsl.OptionalCostInto(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 1)), "", paid),
	} {
		card := dsl.NewArtifact("Invalid", "", dsl.WithAbility(dsl.Activate(dsl.Costs(cost), dsl.GainLife(dsl.You, dsl.Fixed(1)))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	cost := wrap(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 1)))
	for _, card := range []dsl.Card{
		dsl.NewInstant("Invalid resolution", "", dsl.Spell(dsl.IfPaid(dsl.You, dsl.Costs(cost), dsl.GainLife(dsl.You, dsl.FromNumber(paid))))),
		dsl.NewArtifact("Invalid scope", "", dsl.WithAbility(dsl.Activate(dsl.Costs(cost), dsl.GainLife(dsl.You, dsl.Fixed(1)))), dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.LifePayCost(1)), dsl.GainLife(dsl.You, dsl.FromNumber(paid))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid capture scope accepted")
		}
	}
}

func TestOptionalCounterAlternativeResultSpellAdditionalCost(t *testing.T) {
	paid := dsl.BindNumber("paid")
	spell := dsl.NewInstant("Synthetic optional counter spell", "",
		dsl.Spell(dsl.GainLife(dsl.You, dsl.FromNumber(paid))),
		dsl.WithAdditionalCost(dsl.OptionalCostInto(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 0)), "Pay?", paid)))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	a := &optionalXAcceptPlayer{mage.NewBasePlayer("A")}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("absent spell source paid a fixed-counter cost")
	}
}

func TestOptionalCounterAlternativeResultDoesNotDominate(t *testing.T) {
	paid := dsl.BindNumber("paid")
	optional := dsl.OptionalCostInto(dsl.EitherCost(dsl.RemoveCountersCost(core.Charge, 1)), "Pay?", paid)
	card := dsl.NewArtifact("Invalid alternative scope", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.EitherCost(optional, dsl.LifePayCost(1))), dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("conditional result dominated outside its branch")
	}
}
