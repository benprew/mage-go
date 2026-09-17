package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestPreventionShieldNewIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "", 3, 3)
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.AddPreventionShield(card.ID(), 2)
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(card.ID()), 1, uuid.Nil)
	if g.FindPermanent(card.ID()).Damage != 1 {
		t.Fatal("shield followed a returned card")
	}
}

func TestTypedPreventionConsumptionAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, PreventNextDamage(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	ca := clone.GetPlayer(a.PlayerID())
	clone.DealDamageToPlayer(ca, 3, uuid.Nil)
	if ca.Life() != 19 || a.Life() != 20 {
		t.Fatal("cloned shield changed original")
	}
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	g.DealDamageToPlayer(a, 2, uuid.Nil)
	if a.Life() != 19 {
		t.Fatal("shield capacity was not consumed")
	}
	g.effects.RemoveEndOfTurn()
	g.effects.ClearReplacementsEndOfTurn()
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	if a.Life() != 18 {
		t.Fatal("expired shield prevented damage")
	}
}
