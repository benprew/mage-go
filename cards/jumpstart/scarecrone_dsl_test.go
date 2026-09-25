package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestScarecroneSacrificeDraw(t *testing.T) {
	for _, sacrifice := range []string{"Scarecrone", "Scuttlemutt"} {
		t.Run(sacrifice, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Scarecrone")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(owner.PlayerID())
			g.PutOnBattlefield(card, controller.PlayerID())
			g.MutablePermanent(card.ID()).Tapped = true
			if sacrifice == "Scuttlemutt" {
				id := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, sacrifice)
				g.MutablePermanent(id).Tapped = true
			}
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scuttlemutt")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Island")
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(0)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 0, nil); err == nil {
				t.Fatal("unpaid activation must fail")
			}
			if g.FindPermanent(card.ID()) == nil || g.StackSize() != 0 {
				t.Fatal("failed payment changed the battlefield or stack")
			}
			g.ChoosePermanent(gametest.PlayerB, sacrifice)
			controller.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || controller.ManaPool().TotalMana() != 0 || len(controller.Hand()) != 0 {
				t.Fatal("activation must pay mana and sacrifice before the draw resolves")
			}
			if len(g.StackPeek().Targets) != 0 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("sacrifice must not declare a target")
			}
			if sacrifice == "Scarecrone" {
				g.AssertGraveyardCount(gametest.PlayerA, sacrifice, 1)
			} else {
				g.AssertGraveyardCount(gametest.PlayerB, sacrifice, 1)
				if g.FindPermanent(card.ID()) == nil {
					t.Fatal("sacrificing another Scarecrow removed the source")
				}
			}
			g.AssertPermanentCount(gametest.PlayerA, "Scuttlemutt", 1)
			g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
			g.ResolveStack()
			g.AssertHandCount(gametest.PlayerB, "Island", 1)
			if len(owner.Hand()) != 0 || g.StackSize() != 0 {
				t.Fatal("ability must finish and draw only for its stored controller")
			}
		})
	}
}
