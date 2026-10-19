package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCollectionSourcePreventionPaymentsAndVisits(t *testing.T) {
	a := NewBasePlayer("A")
	b := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "{R}", 2, 10), b.PlayerID())
	secondCard := NewCreature("Second", "{R}", 2, 10)
	second := g.PutOnBattlefield(secondCard, b.PlayerID())
	member := BindPermanent("payer")
	effect := ForEachInTurnOrder(MatchingPermanents(IsCreature), member,
		UnlessPaid(ControllerOf(From(member)), Costs(ManaCostOf("{2}{R}")), PreventCombatDamageByEachThisTurn(SingletonPermanent(From(member)))))
	b.ManaPool().Add(core.Red, 1)
	b.ManaPool().Add(core.Colorless, 2)
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().TotalMana() != 0 {
		t.Fatal("payment did not spend full mana")
	}
	clone := g.Clone()
	clone.damage.SetResolvingCombatDamage(true)
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, first.ID())
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, second.ID())
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || a.Life() != 20 {
		t.Fatal("member results or clone rules were shared")
	}
	g.PhaseIn(g.PhaseOut(second.ID()))
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(a, 2, second.ID())
	if a.Life() != 20 {
		t.Fatal("phase cycle lost the saved visit")
	}
	g.RemoveFromBattlefield(second)
	g.PutOnBattlefield(secondCard, b.PlayerID())
	g.DealDamageToPlayer(a, 2, second.ID())
	if a.Life() != 18 {
		t.Fatal("returned visit inherited prevention")
	}
	clone.RunStepWithPriority(core.Cleanup)
	clone.damage.SetResolvingCombatDamage(true)
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, second.ID())
	if clone.GetPlayer(a.PlayerID()).Life() != 16 {
		t.Fatalf("cleanup life=%d want 16", clone.GetPlayer(a.PlayerID()).Life())
	}
}
