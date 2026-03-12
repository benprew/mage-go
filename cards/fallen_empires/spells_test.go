package fallen_empires

import (
	"testing"

	_ "github.com/mage/mage/cards/limited" // register base cards
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestHymnToTourach_DiscardTwoAtRandom(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hymn to Tourach")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hymn to Tourach", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// PlayerB started with 3 Forests in hand, should have 1 left after discarding 2
	g.AssertHandCount(gametest.PlayerB, "Forest", 1)
}

func TestHymnToTourach_EmptyHand(t *testing.T) {
	// If target has fewer than 2 cards, they discard what they can.
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hymn to Tourach")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hymn to Tourach", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// PlayerB had 1 Forest, discards 1 (only 1 available), hand empty.
	g.AssertHandCount(gametest.PlayerB, "Forest", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Forest", 1)
}

func TestIcatianTown_CreateFourTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Icatian Town")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Icatian Town")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Citizen", 4)
}
