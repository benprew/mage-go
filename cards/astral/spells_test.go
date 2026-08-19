package astral

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestCallFromTheGraveUsesASelectedRandomGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Whimsy")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Prismatic Dragon")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Call from the Grave")
	g.SetRandomResults([]int{1, 0})
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Call from the Grave")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Prismatic Dragon", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Prismatic Dragon", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Whimsy", 1)
	g.AssertLife(gametest.PlayerA, 16)
}

func TestCallFromTheGraveDoesNothingWhenSelectedGraveyardHasNoCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Whimsy")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Prismatic Dragon")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Call from the Grave")
	g.SetRandomResults([]int{0})
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Call from the Grave")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Prismatic Dragon", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Prismatic Dragon", 1)
	g.AssertLife(gametest.PlayerA, 20)
}
