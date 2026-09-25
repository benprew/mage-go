package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestBloodHostSourceLifetime(t *testing.T) {
	for _, scenario := range []string{"present", "leaves", "returns"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Blood Host")
			card := g.FindPermanent(source).Card
			card.SetOwner(owner.PlayerID())
			g.FindPermanent(source).Tapped = true
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			controller.ManaPool().Add(core.Black, 2)
			g.ChoosePermanent(gametest.PlayerB, "Grizzly Bears")
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), source, 0, nil); err != nil {
				t.Fatal(err)
			}
			g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
			g.AssertLife(gametest.PlayerB, 20)
			if controller.ManaPool().TotalMana() != 0 || g.FindPermanent(source).Counters[core.P1P1] != 0 {
				t.Fatal("pay mana and sacrifice before the counter and life gain")
			}
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("activation must use the stack without targets")
			}
			if scenario != "present" {
				g.BouncePermanentToHand(g.FindPermanent(source))
				if scenario == "returns" {
					owner.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, owner.PlayerID())
				}
			}
			g.ResolveStack()
			g.AssertLife(gametest.PlayerB, 22)
			g.AssertLife(gametest.PlayerA, 20)
			switch scenario {
			case "present":
				g.AssertCounterCount(gametest.PlayerB, "Blood Host", core.P1P1, 1)
			case "returns":
				g.AssertCounterCount(gametest.PlayerA, "Blood Host", core.P1P1, 0)
			}
		})
	}
}
