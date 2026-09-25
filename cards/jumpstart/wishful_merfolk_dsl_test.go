package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestWishfulMerfolkSourceLifetime(t *testing.T) {
	for _, scenario := range []string{"present", "leaves", "returns"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wishful Merfolk")
			card := g.FindPermanent(source).Card
			player := g.GetPlayer(gametest.PlayerA)
			g.FindPermanent(source).Tapped = true
			player.ManaPool().Add(core.Blue, 2)
			if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 0, nil); err != nil {
				t.Fatal(err)
			}
			if player.ManaPool().TotalMana() != 0 || !g.FindPermanent(source).HasSubType("Merfolk") {
				t.Fatal("pay before changing the source")
			}
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("expected one untargeted ability on the stack")
			}
			if scenario != "present" {
				g.BouncePermanentToHand(g.FindPermanent(source))
				if scenario == "returns" {
					player.RemoveFromHand(source)
					g.PutOnBattlefield(card, player.PlayerID())
				}
			}
			g.ResolveStack()
			g.ApplyContinuousEffects()
			if scenario == "leaves" {
				return
			}
			changed := scenario == "present"
			p := g.FindPermanent(source)
			if p.HasSubType("Human") != changed || p.HasSubType("Merfolk") == changed {
				t.Fatal("only the original source becomes a Human instead of a Merfolk")
			}
			g.AssertHasAbility(gametest.PlayerA, "Wishful Merfolk", core.Defender, !changed)
			g.AssertPowerToughness(gametest.PlayerA, "Wishful Merfolk", 3, 2)
		})
	}
}
