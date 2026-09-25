package antiquities

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestTawnosWandResolution(t *testing.T) {
	for _, change := range []string{"none", "power before", "power after", "returned", "source leaves", "ability removal", "phased"} {
		t.Run(change, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a := g.GetPlayer(gametest.PlayerA)
			sourceID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Wand")
			targetID := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			blockerID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
			target := g.FindPermanent(targetID)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), sourceID, 0, []uuid.UUID{targetID}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(sourceID).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect activation payment")
			}
			switch change {
			case "power before":
				g.AddContinuousEffect(mage.TemporaryBoost(targetID, 1, 0))
				g.ApplyContinuousEffects()
			case "returned":
				card := target.Card
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(card, target.ControllerID())
			case "source leaves":
				g.RemoveFromBattlefield(g.FindPermanent(sourceID))
			case "phased":
				target.PhasedOut = true
			}
			g.ResolveStack()
			if change == "power after" {
				g.AddContinuousEffect(mage.TemporaryBoost(targetID, 1, 0))
			}
			if change == "ability removal" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(targetID, core.EndOfTurn))
			}
			g.ApplyContinuousEffects()
			target = g.FindPermanentIncludingPhased(targetID)
			blocked := mage.CanBlock(g.FindPermanent(blockerID), target, g.Game)
			wantBlocked := change == "power before" || change == "returned" || change == "phased"
			if blocked != wantBlocked {
				t.Fatalf("CanBlock = %v, want %v", blocked, wantBlocked)
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if !mage.CanBlock(g.FindPermanent(blockerID), target, g.Game) {
				t.Fatal("restriction survived cleanup")
			}
		})
	}
}
