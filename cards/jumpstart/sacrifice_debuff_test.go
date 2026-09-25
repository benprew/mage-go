package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestPlaguedRusalkaSacrificesItself(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plagued Rusalka")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Plagued Rusalka", "Hill Giant")
	g.ChoosePermanent(gametest.PlayerA, "Plagued Rusalka")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Plagued Rusalka", 1)
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 2, 2)
}
