package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestGlobalCombatPreventionLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Recipient", "", 4, 4)
	card.SetOwner(b.PlayerID())
	g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, PreventAllCombatDamage(), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.effects.ClearReplacementsEndOfTurn()
	for _, game := range []*Game{g, clone} {
		player := game.GetPlayer(b.PlayerID())
		game.damage.SetResolvingCombatDamage(true)
		game.DealDamageToPlayer(player, 2, uuid.Nil)
		game.DealDamageToPermanent(game.FindPermanent(card.ID()), 2, uuid.Nil)
		game.damage.SetResolvingCombatDamage(false)
		wantLife, wantDamage := 20, 0
		if game == clone {
			wantLife, wantDamage = 18, 2
		}
		if player.Life() != wantLife || game.FindPermanent(card.ID()).Damage != wantDamage {
			t.Fatalf("clone %v: life %d, damage %d; want %d, %d", game == clone, player.Life(), game.FindPermanent(card.ID()).Damage, wantLife, wantDamage)
		}
		game.DealDamageToPlayer(player, 1, uuid.Nil)
		game.DealDamageToPermanent(game.FindPermanent(card.ID()), 1, uuid.Nil)
		if player.Life() != wantLife-1 || game.FindPermanent(card.ID()).Damage != wantDamage+1 {
			t.Fatal("combat prevention stopped noncombat damage")
		}
	}
}
