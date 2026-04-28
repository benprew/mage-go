package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestDraconicRoar_RevealedDragonDealsExtra verifies the optional reveal pays
// off as +3 damage to the targeted creature's controller.
func TestDraconicRoar_RevealedDragonDealsExtra(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertHandCount(gametest.PlayerA, "Shivan Dragon", 1)
}

// TestDraconicRoar_NoDragonNoBonus verifies that without a Dragon to reveal
// or control, only the base 3 damage applies and the controller takes no hit.
func TestDraconicRoar_NoDragonNoBonus(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerB, 20)
}

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
