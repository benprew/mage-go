package jumpstart

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestFeralHydraCounterSourceLifetime(t *testing.T) {
	for _, activator := range []gametest.PlayerRef{gametest.PlayerA, gametest.PlayerB} {
		for _, scenario := range []string{"present", "leaves", "returns"} {
			t.Run(fmt.Sprint(activator)+"/"+scenario, func(t *testing.T) {
				g := gametest.NewTestGame(t)
				source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Feral Hydra")
				card := g.FindPermanent(source).Card
				g.FindPermanent(source).AddCounter(core.P1P1, 2)
				g.FindPermanent(source).Tapped = true
				player := g.GetPlayer(activator)
				player.ManaPool().Add(core.Green, 3)
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 1, nil); err != nil {
					t.Fatal(err)
				}
				if player.ManaPool().TotalMana() != 0 || g.FindPermanent(source).Counters[core.P1P1] != 2 {
					t.Fatal("pay before adding the counter")
				}
				if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 || g.StackPeek().Controller != player.PlayerID() {
					t.Fatal("the activator controls the untargeted ability")
				}
				if scenario != "present" {
					g.BouncePermanentToHand(g.FindPermanent(source))
					if scenario == "returns" {
						g.GetPlayer(gametest.PlayerA).RemoveFromHand(source)
						g.PutOnBattlefield(card, g.GetPlayer(gametest.PlayerA).PlayerID())
						g.FindPermanent(source).AddCounter(core.P1P1, 2)
					}
				}
				g.ResolveStack()
				switch scenario {
				case "present":
					g.AssertCounterCount(gametest.PlayerA, "Feral Hydra", core.P1P1, 3)
				case "returns":
					g.AssertCounterCount(gametest.PlayerA, "Feral Hydra", core.P1P1, 2)
				}
			})
		}
	}
}
