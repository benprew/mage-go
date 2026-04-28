package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestExplore_AllowsSecondLand verifies that after Explore resolves, the
// active player's land-play allowance is bumped to 2. The harness's auto-
// land-play loop then plays both Forests in the same main phase.
func TestExplore_AllowsSecondLand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Explore")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Explore")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Forest", 4)
}
