package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCounterPaymentLockPublicDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Synthetic counter payment", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.RemoveCountersCost(core.Charge, 2)),
		dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a := mage.NewBasePlayer("A")
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(card.ID()).AddCounter(core.Charge, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()).Counters[core.Charge] != 0 {
		t.Fatal("counters were not paid at activation")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("resolution crossed clone boundary")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("paid activation did not resolve")
	}
}
