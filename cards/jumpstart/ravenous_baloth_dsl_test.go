package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRavenousBalothSacrificeChoice(t *testing.T) {
	for _, sacrifice := range []string{"Ravenous Baloth", "Rumbling Baloth"} {
		t.Run(sacrifice, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Ravenous Baloth")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(owner.PlayerID())
			g.PutOnBattlefield(card, controller.PlayerID())
			g.MutablePermanent(card.ID()).Tapped = true
			if sacrifice == "Rumbling Baloth" {
				id := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, sacrifice)
				g.MutablePermanent(id).Tapped = true
			}
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rumbling Baloth")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			g.SetLife(gametest.PlayerA, 20)
			g.SetLife(gametest.PlayerB, 20)
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(0)
			g.ChoosePermanent(gametest.PlayerB, sacrifice)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 {
				t.Fatal("activation must use the stack")
			}
			g.AssertLife(gametest.PlayerB, 20)
			if len(g.StackPeek().Targets) != 0 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("sacrifice must not declare a target")
			}
			if sacrifice == "Ravenous Baloth" {
				g.AssertGraveyardCount(gametest.PlayerA, sacrifice, 1)
			} else {
				g.AssertGraveyardCount(gametest.PlayerB, sacrifice, 1)
				if g.FindPermanent(card.ID()) == nil {
					t.Fatal("sacrificing another Beast removed the source")
				}
			}
			g.AssertPermanentCount(gametest.PlayerA, "Rumbling Baloth", 1)
			g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
			g.ResolveStack()
			g.AssertLife(gametest.PlayerB, 24)
			g.AssertLife(gametest.PlayerA, 20)
		})
	}
}
