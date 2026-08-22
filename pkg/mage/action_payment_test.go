package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPayAttackCosts_ReservesAttackerFromManaPayment(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()

	attackerCard := NewCreature("Mana Attacker", "{G}", 1, 1, WithManaAbility(Green))
	attackerCard.SetOwner(playerID)
	attacker := g.PutOnBattlefield(attackerCard, playerID)
	attacker.RevokeBaseAttr(AttrSummonSick)
	firstLand := addLand(t, g, playerID, "Forest 1", Green)
	secondLand := addLand(t, g, playerID, "Forest 2", Green)
	g.effects.AddAttackCost(attacker.ID(), GenericCost(3))

	if g.PayAttackCosts(attacker, playerID, false) {
		t.Fatal("attack cost should be unaffordable without tapping the attacker")
	}
	if attacker.Tapped || firstLand.Tapped || secondLand.Tapped {
		t.Fatalf("failed attack payment changed tapped state: attacker=%v first=%v second=%v", attacker.Tapped, firstLand.Tapped, secondLand.Tapped)
	}
	if got := g.players[0].ManaPool().TotalMana(); got != 0 {
		t.Fatalf("failed attack payment left %d mana in the pool", got)
	}
}

func TestPayAttackCosts_AggregatePaymentIsAtomic(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()

	attackerCard := NewCreature("Taxed Attacker", "{1}{G}", 2, 2)
	attackerCard.SetOwner(playerID)
	attacker := g.PutOnBattlefield(attackerCard, playerID)
	attacker.RevokeBaseAttr(AttrSummonSick)
	lands := make([]*Permanent, 0, 5)
	for range 5 {
		lands = append(lands, addLand(t, g, playerID, "Forest", Green))
	}
	g.effects.AddAttackCost(attacker.ID(), GenericCost(3))
	g.effects.AddAttackCost(attacker.ID(), GenericCost(3))

	if g.PayAttackCosts(attacker, playerID, false) {
		t.Fatal("two {3} attack costs should be unaffordable with five mana")
	}
	for i, land := range lands {
		if land.Tapped {
			t.Fatalf("failed aggregate payment tapped land %d", i)
		}
	}
	if got := g.players[0].ManaPool().TotalMana(); got != 0 {
		t.Fatalf("failed aggregate payment left %d mana in the pool", got)
	}
}

func TestActivateAbility_TotalCostFailureIsAtomic(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	playerID := player.PlayerID()
	player.SetLife(3)

	sourceCard := NewArtifact("Expensive Device", "{1}",
		WithActivatedAbility(
			GainLife(1),
			GenericCost(1),
			WithCost(LifePayCost(2)),
			WithCost(LifePayCost(2)),
		),
	)
	sourceCard.SetOwner(playerID)
	source := g.PutOnBattlefield(sourceCard, playerID)
	land := addLand(t, g, playerID, "Forest", Green)

	if err := g.ActivateAbilityByIndex(playerID, source.ID(), 0, nil); err == nil {
		t.Fatal("activation should fail when its total life payment is unaffordable")
	}
	if land.Tapped {
		t.Fatal("failed activation tapped a mana source")
	}
	if got := player.Life(); got != 3 {
		t.Fatalf("failed activation changed life total to %d", got)
	}
	if got := player.ManaPool().TotalMana(); got != 0 {
		t.Fatalf("failed activation left %d mana in the pool", got)
	}
}

func TestCostPaymentValidationDoesNotConsumeManaAbilityActivationLimit(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	playerID := player.PlayerID()

	sourceCard := NewArtifact("Limited Mana Source", "{1}",
		WithActivatedAbility(
			AddMana(Colorless, 1),
			Tap(),
			WithActivationLimit(ActivationLimits{OncePerTurn: true}),
		),
	)
	sourceCard.SetOwner(playerID)
	source := g.PutOnBattlefield(sourceCard, playerID)

	if !g.TryPayMana(playerID, "{1}") {
		t.Fatal("validation consumed the mana ability's once-per-turn activation")
	}
	if !source.Tapped {
		t.Fatal("successful payment did not tap its mana source")
	}
	if got := player.ManaPool().TotalMana(); got != 0 {
		t.Fatalf("successful payment left %d mana in the pool", got)
	}
}
