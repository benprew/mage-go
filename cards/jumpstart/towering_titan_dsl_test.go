package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestToweringTitanSacrificeTrample(t *testing.T) {
	for _, sacrifice := range []string{"Towering Titan", "Wall of Wood"} {
		t.Run(sacrifice, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wood")
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Towering Titan")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(owner.PlayerID())
			g.PutOnBattlefield(card, controller.PlayerID())
			if sacrifice == "Towering Titan" {
				g.AddContinuousEffect(mage.TemporaryKeyword(card.ID(), core.Defender))
				g.ApplyContinuousEffects()
			} else {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Wood")
			}
			g.FindPermanent(card.ID()).Tapped = true
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(0)
			g.ChoosePermanent(gametest.PlayerB, sacrifice)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("activation must use the stack without targets")
			}
			sacrificeOwner := gametest.PlayerB
			if sacrifice == "Towering Titan" {
				sacrificeOwner = gametest.PlayerA
			}
			g.AssertGraveyardCount(sacrificeOwner, sacrifice, 1)
			for _, p := range g.Battlefield {
				if p.HasKeyword(core.Trample) {
					t.Fatal("trample was granted before resolution")
				}
			}
			g.ResolveStack()
			g.ApplyContinuousEffects()
			for _, p := range g.Battlefield {
				if p.HasType(core.TypeCreature) && !p.HasKeyword(core.Trample) {
					t.Fatal("each player's creatures must gain trample")
				}
			}
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
			for _, p := range g.Battlefield {
				if p.Name() == "Hill Giant" && p.HasKeyword(core.Trample) {
					t.Fatal("a later creature must not gain trample")
				}
			}
			g.AssertPermanentCount(gametest.PlayerA, "Wall of Wood", 1)
			g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		})
	}
}
