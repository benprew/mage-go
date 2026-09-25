package limited

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestDwarvenDemolitionTeamNoncreatureWall(t *testing.T) {
	const wall = "Test Dwarven Demolition Team Noncreature Wall"
	dsl.Register(wall, func() dsl.Card {
		return dsl.NewEnchantment(wall, "{1}", dsl.WithSubTypes("Wall"))
	})

	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Demolition Team")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, wall)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Demolition Team", wall)
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Dwarven Demolition Team", 1)
	g.AssertPermanentCount(gametest.PlayerB, wall, 0)
	g.AssertGraveyardCount(gametest.PlayerB, wall, 1)
}
