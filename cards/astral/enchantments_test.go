package astral

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestNecropolisOfAzarCountsNonblackCreatureDeaths(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Necropolis of Azar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Terror")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Terror", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertCounterCount(gametest.PlayerA, "Necropolis of Azar", core.Husk, 1)
}

func TestNecropolisOfAzarCreatesRandomSpawn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Necropolis of Azar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 5)
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Necropolis of Azar", core.Husk, 1)
	g.SetRandomResults([]int{0, 2})
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Necropolis of Azar")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertCounterCount(gametest.PlayerA, "Necropolis of Azar", core.Husk, 0)
	g.AssertPermanentCount(gametest.PlayerA, "Spawn of Azar", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Spawn of Azar", 1, 3)
	g.AssertHasColor(gametest.PlayerA, "Spawn of Azar", core.Black, true)
	g.AssertHasAbility(gametest.PlayerA, "Spawn of Azar", core.Swampwalk, true)
}
