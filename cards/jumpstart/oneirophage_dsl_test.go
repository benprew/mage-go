package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestOneirophageDrawSourceLifetime(t *testing.T) {
	for _, scenario := range []string{"present", "leaves", "returns before placement", "returns on stack"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Oneirophage")
			card := g.FindPermanent(source).Card
			controller := g.GetPlayer(gametest.PlayerB).PlayerID()
			card.SetOwner(g.GetPlayer(gametest.PlayerA).PlayerID())
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Mountain")
			g.PlayerDrawCard(g.GetPlayer(gametest.PlayerA))
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("the owner's draw must not trigger a creature controlled by the other player")
			}
			g.PlayerDrawCard(g.GetPlayer(gametest.PlayerB))
			returnSource := func() {
				g.BouncePermanentToHand(g.FindPermanent(source))
				g.GetPlayer(gametest.PlayerA).RemoveFromHand(source)
				g.PutOnBattlefield(card, controller)
			}
			if scenario == "returns before placement" {
				returnSource()
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.StackPeek().Controller != controller || len(g.StackPeek().Targets) != 0 {
				t.Fatalf("want one untargeted draw trigger for %v; stack: %+v", controller, g.StackObjects())
			}
			if g.FindPermanent(source).Counters[core.P1P1] != 0 {
				t.Fatal("the counter must wait for resolution")
			}
			switch scenario {
			case "leaves":
				g.BouncePermanentToHand(g.FindPermanent(source))
			case "returns on stack":
				returnSource()
			}
			g.ResolveStack()
			if scenario == "leaves" {
				g.AssertHandCount(gametest.PlayerA, "Oneirophage", 1)
				return
			}
			want := 0
			if scenario == "present" {
				want = 1
			}
			g.AssertCounterCount(gametest.PlayerB, "Oneirophage", core.P1P1, want)
		})
	}
}
