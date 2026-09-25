package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestBubblingCauldronSacrificeChoice(t *testing.T) {
	for _, sacrifice := range []string{"Bubbling Cauldron", "Grizzly Bears"} {
		t.Run(sacrifice, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Bubbling Cauldron")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(owner.PlayerID())
			g.PutOnBattlefield(card, controller.PlayerID())
			if sacrifice == "Bubbling Cauldron" {
				g.AddContinuousEffect(mage.TemporaryAnimate(card.ID(), 2, 2))
				g.AddContinuousEffect(mage.TemporaryKeyword(card.ID(), core.Haste))
				g.ApplyContinuousEffects()
			} else {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerB, sacrifice)
			}
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Arcane Encyclopedia")
			g.SetLife(gametest.PlayerA, 20)
			g.SetLife(gametest.PlayerB, 10)
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(0)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 0, nil); err == nil {
				t.Fatal("unpaid activation must fail")
			}
			if g.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("failed payment changed the battlefield or stack")
			}
			g.ChoosePermanent(gametest.PlayerB, sacrifice)
			controller.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || controller.ManaPool().TotalMana() != 0 {
				t.Fatal("activation must pay costs before stack resolution")
			}
			g.AssertLife(gametest.PlayerB, 10)
			if len(g.StackPeek().Targets) != 0 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("sacrifice must not declare a target")
			}
			if sacrifice == "Bubbling Cauldron" {
				g.AssertGraveyardCount(gametest.PlayerA, sacrifice, 1)
			} else {
				g.AssertGraveyardCount(gametest.PlayerB, sacrifice, 1)
				g.AssertTapped(gametest.PlayerB, "Bubbling Cauldron", true)
			}
			g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
			g.AssertPermanentCount(gametest.PlayerB, "Arcane Encyclopedia", 1)
			g.ResolveStack()
			g.AssertLife(gametest.PlayerB, 14)
			g.AssertLife(gametest.PlayerA, 20)
		})
	}
}
