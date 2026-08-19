package astral

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestWhimsy_EachRandomActionCanBeSelected(t *testing.T) {
	actions := []string{
		"bounce", "untap", "tap", "damage four", "draw three", "destroy artifact and gain life",
		"destroy artifact or enchantment", "gain three", "prevent three", "destroy creature or land",
		"mill two", "Wasp", "destroy all", "Suleiman", "Pandora", "discard", "Fog", "Sindbad",
	}
	for action, name := range actions {
		t.Run(fmt.Sprintf("%02d_%s", action+1, name), func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneHand, gametest.PlayerA, "Whimsy")
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Whimsy", 6)
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Whimsy", 6)
			g.SetRandomResults([]int{action, 0, 0, 0, 0})
			g.SetCoinFlipResults([]bool{true, false})
			g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Whimsy", 1)
			g.StopAt(1, core.BeginCombat)
			g.Execute()
		})
	}
}

func TestWhimsy_PerformsXIndependentActionsInOrder(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Whimsy")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Whimsy", 6)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Whimsy", 6)
	g.SetRandomResults([]int{7, 0, 7, 1, 7, 0})
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Whimsy", 3)
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 26)
	g.AssertLife(gametest.PlayerB, 23)
}

func TestWhimsy_NoEligiblePermanentIsSafe(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Whimsy")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Whimsy", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Whimsy", 3)
	g.SetRandomResults([]int{0, 1, 2, 5, 6, 9})
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Whimsy", 6)
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 20)
	g.AssertLife(gametest.PlayerB, 20)
}
