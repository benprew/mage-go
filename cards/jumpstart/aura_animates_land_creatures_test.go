package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestAwakenerDruid_AnimatesTargetForest(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Awakener Druid")
	g.ChooseTarget(gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Awakener Druid")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Forest", 4, 5)
}
